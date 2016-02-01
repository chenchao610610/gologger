package woolog

import (
	"os"
	"runtime"
	"testing"
	"time"
)

func TestLog(t *testing.T) {
	f, e := os.OpenFile("test.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, os.FileMode(0766))
	if e != nil {
		t.Log(e)
		t.Fail()
	}

	l := NewLog()
	l.SetLevel(ALL)
	l.SetOutput(f)

	length := 100
	context := "1234567890-=!@#$%^&*(qwertyuiop[]';lkjhgfdsazxcvbnm,./?><:}{)_+"
	for len(context) < length {
		context += context
	}
	context = string([]byte(context)[:length])

	forever := make(chan int64, runtime.NumCPU())
	t.Log("cpu:", runtime.NumCPU(), "context:", len(context))

	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			s := time.Now()
			for i := 0; i < 12000000; i++ {
				l.Info(context)
			}
			forever <- time.Now().Sub(s).Nanoseconds() / 1000
		}()
	}

	for i := 0; i < runtime.NumCPU(); i++ {
		t.Log(<-forever)
	}
	f.Close()
}
