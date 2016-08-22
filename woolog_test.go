package woolog

import (
	//"os"
	//"runtime"

	"testing"

	//"time"

	//	"log"
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

	//	context := parpareData()

	Error("123ddddddddddd14")
	//log.Println("loglog")
	t.Log("123")
	//	log.SetOutput(os.Stdout)
	//	log.SetFlags(log.Lshortfile)
	//	log.Println("ppppppp")

}
