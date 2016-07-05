package woolog

import (
	"runtime"
	"testing"
	"time"
)

func parpareData() string {
	length := 500
	context := `1234567890-=!@#$%^&*(qwertyuiop[]';lkjhgfdsazxcvbnm,./?><:}{)_+`
	for len(context) < length {
		context += context
	}
	context = string([]byte(context)[:length])
	return context
}

func TestLog(t *testing.T) {
	defer Sync()
	SetLevel(DEBUG)
	SetLogName("/tmp/woolog.log")

	context := parpareData()
	t.Log("cpu:", runtime.NumCPU(), "context:", len(context))

	forever := make(chan int64, runtime.NumCPU())
	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			s := time.Now()
			for i := 0; i < 1000000; i++ {
				Info(context)
			}
			forever <- time.Now().Sub(s).Nanoseconds() / 1000
		}()
	}

	for i := 0; i < runtime.NumCPU(); i++ {
		t.Log(<-forever)
	}
}
