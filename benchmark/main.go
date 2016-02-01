package main

import (
	"bytes"
	"fmt"
	"io"
	"jpushcomm/proto"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
	"woolog"
)

type BW struct {
	FileName string
}

func (this BW) Write(buff []byte) (int, error) {
	f, e := os.OpenFile(this.FileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, os.FileMode(0766))
	if e != nil {
		return 0, e
	}
	defer f.Close()
	return f.Write(buff)
}

func benchio() {
	w := new(bytes.Buffer)
	for i := 0; i < 40960; i++ {
		w.WriteByte('a')
	}

	f, e := os.OpenFile("benchio.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, os.FileMode(0766))
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	defer f.Close()

	buff := w.Bytes()

	s := time.Now()
	for i := 0; i < 20000; i++ {
		f.Write(buff)
	}
	fmt.Println("benchio", time.Now().Sub(s).Nanoseconds()/1000)
	time.Sleep(3 * time.Second)
}

func benchlog(chioce int) {
	fmt.Println("chioce:", chioce)
	var w io.Writer

	if chioce == 1 {
		f, e := os.OpenFile("benchlog.1.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, os.FileMode(0766))
		if e != nil {
			fmt.Println(e.Error())
			return
		}
		defer f.Close()
		w = f
	} else if chioce == 2 {
		f := BW{FileName: "benchlog.2.log"}
		w = f
	}

	woolog.SetOutput(w)
	woolog.SetLevel(woolog.DEBUG)

	length := 100
	context := "1234567890-=!@#$%^&*(qwertyuiop[]';lkjhgfdsazxcvbnm,./?><:}{)_+"
	for len(context) < length {
		context += context
	}
	context = string([]byte(context)[:length])

	c := 1200000
	fmt.Println("cpu:", runtime.NumCPU(), "len:", len(context), "count:", c, "per:", c/runtime.NumCPU())

	forever := make(chan int64, runtime.NumCPU())
	head := proto.Head{}
	fmt.Println(head)

	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			s := time.Now()
			for i := 0; i < c/runtime.NumCPU(); i++ 
				woolog.Info(fmt.Sprintf("sxx:%d uid:%d head%d %d %d %d %d",
					1000002,
					10000000,
					head.Command,
					head.SID,
					head.Len,
					head.RID,
					head.Version))
			}
			forever <- time.Now().Sub(s).Nanoseconds() / 1000
		}()
	}

	for i := 0; i < runtime.NumCPU(); i++ {
		fmt.Println("benchlog", <-forever)
	}
	time.Sleep(2 * time.Second)
}

func benchStdLog(chioce int) {
	fmt.Println("chioce:", chioce)
	var w io.Writer

	f, e := os.OpenFile("benchStdLog.3.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, os.FileMode(0766))
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	defer f.Close()
	w = f

	l := log.New(w, "INFO", log.LstdFlags|log.Lshortfile)

	length := 100
	context := "1234567890-=!@#$%^&*(qwertyuiop[]';lkjhgfdsazxcvbnm,./?><:}{)_+"
	for len(context) < length {
		context += context
	}
	context = string([]byte(context)[:length])

	c := 1200000
	fmt.Println("cpu:", runtime.NumCPU(), "len:", len(context), "count:", c, "per:", c/runtime.NumCPU())

	forever := make(chan int64, runtime.NumCPU())
	head := proto.Head{}
	fmt.Println(head)

	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			s := time.Now()
			for i := 0; i < c/runtime.NumCPU(); i++ {

				l.Println(fmt.Sprintf("sxx:%d uid:%d head%d %d %d %d %d",
					1000002,
					10000000,
					head.Command,
					head.SID,
					head.Len,
					head.RID,
					head.Version))

			}
			forever <- time.Now().Sub(s).Nanoseconds() / 1000
		}()
	}

	for i := 0; i < runtime.NumCPU(); i++ {
		fmt.Println("benchlog", <-forever)
	}
	time.Sleep(2 * time.Second)
}

func main() {
	f, e := os.Create("profile_file")
	if e != nil {
		fmt.Println(e)
		return
	}
	pprof.StartCPUProfile(f)
	benchio()
	benchlog(1)
	benchlog(2)
	benchStdLog(3)
	pprof.StopCPUProfile()
}
