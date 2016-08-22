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

const (
	DEBUG int = 1
	INFO  int = 2
	WARN  int = 3
	ERROR int = 4
	FATAL int = 5
)

type _stream struct {
	path string
	buff bytes.Buffer
}

type Log struct {
	level     int
	trace     map[uintptr]string
	traceLock sync.Mutex

	capacity     int
	lastDay      int
	pid          string
	out          io.Writer
	logPrePath   string
	logFullPath  string
	logPathMutex sync.Mutex

	ioCh         chan *_stream
	bufferCh     chan *_stream
	bufferChSize int
}

func (this *Log) GetLevel() int {
	return this.level
}

func (this *Log) SetLevel(lev int) {
	this.level = lev
}

func (this *Log) SetLogName(prePath string) {
	this.changeLogPath(prePath)
}

func (this *Log) Sync() {
	for i := 0; i < this.bufferChSize; i++ {
		this.ioCh <- (<-this.bufferCh)
	}
}

func NewLog(logname string, capacity, poolsize int) *Log {
	if capacity <= 0 {
		panic("illegal capacity")
	}

	pid := strconv.Itoa(os.Getpid())

	bufferCh := make(chan *_stream, poolsize)
	ioch := make(chan *_stream, poolsize)

	for i := 0; i < poolsize; i++ {
		bufferCh <- new(_stream)
	}

	l := &Log{
		level:        INFO,
		capacity:     capacity,
		pid:          pid,
		ioCh:         ioch,
		bufferCh:     bufferCh,
		bufferChSize: poolsize,
		trace:        make(map[uintptr]string, 100),
	}

	l.SetLogName(logname)

	go func() {
		l.lookupIO()
	}()

	return l
}

func (this *Log) output(prefix string, pc uintptr, v ...interface{}) {
	this.traceLock.Lock()
	s, ok := this.trace[pc]
	this.traceLock.Unlock()
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

		s = fmt.Sprintf("%s:%d", f, l)

		this.traceLock.Lock()
		this.trace[pc] = s
		this.traceLock.Unlock()
	}

	stream := <-this.bufferCh // TODO: 先阻塞 当io不足有助于限制内存，防止猛涨
	w := &stream.buff
	// time format
	t := time.Now() // TODO: 时间在什么位置取的问题
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

	if day != this.lastDay {
		this.changeLogPath(this.logPrePath)
		this.lastDay = day
	}

	stream.path = this.getFullPath()
	this.ioCh <- stream
}

func (this *Log) getFullPath() string {
	this.logPathMutex.Lock()
	defer this.logPathMutex.Unlock()
	return this.logFullPath
}

func (this *Log) changeLogPath(prePath string) {
	this.logPathMutex.Lock()
	defer this.logPathMutex.Unlock()
	subfix := time.Now().Format("20060102")
	this.logPrePath = prePath
	this.logFullPath = prePath + "." + subfix
}

func (this *Log) lookupIO() {
	lastStream := new(_stream)
	tick := time.Tick(500 * time.Millisecond)

	for {
		select {
		case s := <-this.ioCh:
			if s.path == lastStream.path {
				if s.buff.Len()+lastStream.buff.Len() > this.capacity {
					if lastStream.buff.Len() > 0 {
						writeTpFile(lastStream)
						lastStream.buff.Reset()
					}

					if s.buff.Len() > this.capacity {
						writeTpFile(s)
					} else {
						lastStream.buff.Write(s.buff.Bytes())
					}
				} else {
					lastStream.buff.Write(s.buff.Bytes())
				}
			} else {
				if lastStream.buff.Len() > 0 {
					writeTpFile(lastStream)
					lastStream.buff.Reset()
				}
				writeTpFile(s)
				lastStream.path = s.path
			}
			s.buff.Reset()
			this.bufferCh <- s
		case <-tick:
			if lastStream.buff.Len() > 0 {
				writeTpFile(lastStream)
				lastStream.buff.Reset()
			}
		}
	}
}

func writeTpFile(s *_stream) {
	if s.buff.Len() > 0 {
		defer func() {
			err := recover()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}()
		f, e := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_RDWR, os.FileMode(0766))
		if e != nil {
			return
		}
		defer f.Close()
		f.Write(s.buff.Bytes())
	}
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
	if logobj.level <= DEBUG {
		pc := make([]uintptr, 1)
		runtime.Callers(2, pc) // 层次越深 性能越差
		logobj.output("DEBUG", pc[0], v...)
	}
}

func Info(v ...interface{}) {
	if logobj.level <= INFO {
		pc := make([]uintptr, 1)
		runtime.Callers(2, pc)
		logobj.output("INFO", pc[0], v...)
	}
}

func Warn(v ...interface{}) {
	if logobj.level <= WARN {
		pc := make([]uintptr, 1)
		runtime.Callers(2, pc)
		logobj.output("WARN", pc[0], v...)
	}
}

func Error(v ...interface{}) {
	if logobj.level <= FATAL {
		pc := make([]uintptr, 1)
		runtime.Callers(2, pc)
		logobj.output("FATAL", pc[0], v...)
	}
}

func Fatal(v ...interface{}) {
	if logobj.level <= FATAL {
		pc := make([]uintptr, 1)
		runtime.Callers(2, pc)
		logobj.output("FATAL", pc[0], v...)
	}
}

func Sync() {
	logobj.Sync()
}

//type fileWriter struct {
//	io.Writer
//	FileName string
//}

//func (this fileWriter) Write(buff []byte) (int, error) {
//	f, e := os.OpenFile(this.FileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, os.FileMode(0766))
//	if e != nil {
//		return 0, e
//	}
//	defer f.Close()
//	return f.Write(buff)
//}

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

func init() {
	logobj = NewLog("", 4096, 4)
}
