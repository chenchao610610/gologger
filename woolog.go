package woolog

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

type fileWriter struct {
	io.Writer
	FileName string
}

func (this fileWriter) Write(buff []byte) (int, error) {
	f, e := os.OpenFile(this.FileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, os.FileMode(0766))
	if e != nil {
		return 0, e
	}
	defer f.Close()
	return f.Write(buff)
}

func itoa(w *bytes.Buffer, i int, wid int) {
	// 抄自 /src/log/log.go
	// Assemble decimal in reverse order.
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	w.Write(b[bp:])
}

const (
	DEBUG int = 1
	INFO  int = 2
	WARN  int = 3
	ERROR int = 4
	FATAL int = 5
)

type Log struct {
	level     int
	out       io.Writer
	trace     map[uintptr]string
	traceLock sync.Mutex
	// time goid info args
	memch     chan *bytes.Buffer
	ioch      chan *bytes.Buffer
	unusech   chan *bytes.Buffer
	capacity  int
	pid       string
	logname   string
	logsubfix string
	logmutex  sync.Mutex
}

func (this *Log) output(prefix string, pc uintptr, v ...interface{}) {
	t := time.Now() // TODO: 时间在什么位置取的问题
	s, ok := this.trace[pc]
	if !ok {
		f, l := runtime.FuncForPC(pc).FileLine(pc)
		// 抄自 /src/log/log.go
		c := 0
		for i := len(f) - 1; i > 0; i-- {
			if f[i] == '/' {
				c += 1
				if 2 == c {
					f = f[i+1:]
					break
				}
			}
		}
		s = fmt.Sprintf("%s:%d", f, l-1)
		func() {
			this.traceLock.Lock()
			defer this.traceLock.Unlock()
			this.trace[pc] = s
		}()
	}

	w := <-this.memch // TODO: 先阻塞 当io不足有助于限制内存，防止猛涨

	// time format
	year, month, day := t.Date()
	itoa(w, year, 4)
	w.WriteByte('/')
	itoa(w, int(month), 2)
	w.WriteByte('/')
	itoa(w, day, 2)
	w.WriteByte(' ')

	hour, min, sec := t.Clock()
	itoa(w, hour, 2)
	w.WriteByte(':')
	itoa(w, min, 2)
	w.WriteByte(':')
	itoa(w, sec, 2)
	w.WriteByte('.')
	itoa(w, t.Nanosecond()/1e3, 6)
	w.WriteByte(' ')

	//pid
	w.WriteString(this.pid)
	w.WriteByte(' ')

	// file:line
	w.WriteString(s)
	w.WriteByte(' ')

	// level
	w.WriteString(prefix)
	w.WriteByte(' ')

	fmt.Fprintln(w, v...)

	if w.Len() < this.capacity {
		this.memch <- w
	} else {
		this.ioch <- w
		this.memch <- (<-this.unusech)
	}
}

func (this *Log) lookupMem() {
	for {
		t := time.After(1 * time.Second)
		<-t
		this.checkOutputObject()

		w := <-this.memch
		if w.Len() > 0 {
			this.memch <- (<-this.unusech)
			this.ioch <- w
		} else {
			this.memch <- w
		}
	}
}

func (this *Log) checkOutputObject() {
	this.logmutex.Lock()
	defer this.logmutex.Unlock()

	if len(this.logname) > 0 {
		subfix := time.Now().Format("20060102")
		if this.logsubfix != subfix {
			this.out = fileWriter{FileName: (this.logname + "." + subfix)}
		}
	} else {
		if this.out == nil {
			this.out = os.Stdout
		}
	}
}

func (this *Log) lookupIO() {
	for {
		select {
		case w := <-this.ioch:
			if w.Len() > 0 {
				this.out.Write(w.Bytes())
				w.Reset()
				this.unusech <- w
			} else {
				this.unusech <- w
			}
		}
	}
}

func (this *Log) GetLevel() int {
	return this.level
}

func (this *Log) SetLevel(lev int) {
	this.level = lev
}

func (this *Log) SetLogName(logname string) {
	this.logmutex.Lock()
	this.logname = logname
	this.logsubfix = ""
	this.logmutex.Unlock()
	this.checkOutputObject()
}

func (this *Log) Sync() {
	for i := 0; i < runtime.NumCPU()+2; i++ {
		this.ioch <- (<-this.memch)
		this.memch <- (<-this.unusech)
	}
}

func NewLog(logname string) *Log {
	l := &Log{
		level:    INFO,
		capacity: 40000,
		pid:      strconv.Itoa(os.Getpid()),
		trace:    make(map[uintptr]string, 100)}
	l.SetLogName(logname)

	c := runtime.NumCPU() + 2
	l.memch = make(chan *bytes.Buffer, 1)
	l.ioch = make(chan *bytes.Buffer, c)
	l.unusech = make(chan *bytes.Buffer, c)

	for i := 0; i < c; i++ {
		w := new(bytes.Buffer)
		w.Grow(l.capacity + 960)
		l.unusech <- w
	}
	l.memch <- (<-l.unusech)

	go l.lookupIO()
	go l.lookupMem()
	return l
}

var logobj *Log

func SetLogName(logname string) {
	logobj.SetLogName(logname)
}

func GetLevel() int {
	return logobj.GetLevel()
}

func SetLevel(lev int) {
	logobj.SetLevel(lev)
}

func Debug(v ...interface{}) {
	if logobj.level <= DEBUG && logobj.out != nil {
		pc := make([]uintptr, 1)
		runtime.Callers(2, pc) // 层次越深 性能越差
		logobj.output("DEBUG", pc[0], v...)
	}
}

func Info(v ...interface{}) {
	if logobj.level <= INFO && logobj.out != nil {
		pc := make([]uintptr, 1)
		runtime.Callers(2, pc)
		logobj.output("INFO", pc[0], v...)
	}
}

func Warn(v ...interface{}) {
	if logobj.level <= WARN && logobj.out != nil {
		pc := make([]uintptr, 1)
		runtime.Callers(2, pc)
		logobj.output("WARN", pc[0], v...)
	}
}

func Error(v ...interface{}) {
	if logobj.level <= FATAL && logobj.out != nil {
		pc := make([]uintptr, 1)
		runtime.Callers(2, pc)
		logobj.output("FATAL", pc[0], v...)
	}
}

func Fatal(v ...interface{}) {
	if logobj.level <= FATAL && logobj.out != nil {
		pc := make([]uintptr, 1)
		runtime.Callers(2, pc)
		logobj.output("FATAL", pc[0], v...)
	}
}

func Sync() {
	logobj.Sync()
}

func init() {
	logobj = NewLog("")
}
