package firefly

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"path/filepath"
	"strings"
	"testing"
)

func Test_array(t *testing.T) {
	var m = make(map[string][]*FlvFileInfo)

	month := "2021-08"

	path := "D:\\work-go\\monibuca\\resource\\live"
	if files, err := tree(path, 0); err == nil {
		for i := 0; i < len(files); i++ {
			f := files[i]
			curTime := getFileDate(f.Path)
			if strings.Contains(curTime, month) {
				array, _ := m[curTime]
				array = append(array, f)
				m[curTime] = array
			}
		}
	}

	if s, err := json.MarshalIndent(m, "", "\t"); err == nil {
		log.Println(string(s))
	}
}

func Test_file(t *testing.T) {
	filePaths := "D:\\work-go\\monibuca\\resource\\live\\hk\\2021-08\\08-22-181514.flv"

	p := strings.TrimPrefix(filePaths, config.SavePath)
	p = strings.ReplaceAll(p, "\\", "/")

	filename2 := filepath.Base(p)
	dir := filepath.Dir(p)

	start := strings.LastIndex(dir, "\\") + 1
	parentDir := string([]byte(dir)[start : len(dir)-2])

	days := string([]byte(filename2)[:5])
	curTime := parentDir + days

	str := "2021-08"
	log.Printf("curTime = %s", curTime)

	if strings.Contains(curTime, str) {
		t.Log("pass")
	} else {
		t.Error("fail")
	}

}

func Test_StringHex(t *testing.T) {
	a := "0123456789"
	bytes, err := hex.DecodeString(a)
	if err != nil {
		t.Error(err)
	}

	t.Log(string(bytes))

}
