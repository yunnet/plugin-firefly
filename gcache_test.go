package firefly

import (
	"fmt"
	"github.com/bluele/gcache"
	"os"
	"strings"
	"testing"
)

func Test_gCache(t *testing.T) {
	gc := gcache.New(20).LRU().Build()

	path := "D:\\mnt\\sd\\record\\live\\hw\\2021-09-28\\18-14-27.mp4"
	path = strings.ReplaceAll(path, "\\", "/")

	var f *os.File
	f, err := os.Open(path)
	if err != nil {
		t.Log(err)
		return
	}
	defer f.Close()

	fileInfo, err := f.Stat()
	if err != nil {
		t.Log(err)
		return
	}

	var recInfo *RecFileInfo

	key := fileInfo.Name()
	t.Log("key = " + key)

	value, err := gc.Get(key)
	if err != nil {
		recInfo = &RecFileInfo{
			Url:       strings.TrimPrefix(path, "/"),
			Size:      fileInfo.Size(),
			Timestamp: getMp4Timestamp(path).Unix(),
			Duration:  GetMP4Duration(f),
		}
		gc.Set(key, recInfo)
	} else {
		recInfo, _ = (value).(*RecFileInfo)
	}
	rec := recInfo.String()

	fmt.Println(rec)

	value2, err := gc.Get(key)
	if err != nil {
		t.Log(err)
		return
	}
	recInfo, _ = (value2).(*RecFileInfo)
	fmt.Println(recInfo.String())

}
