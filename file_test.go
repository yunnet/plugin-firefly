package firefly

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func Test_days(t *testing.T) {
	streamPath := "resource"
	y := time.Now().Format("2006/01/02/150405")
	path := filepath.Join(streamPath, y) + ".flv"
	fmt.Println(path)
}

func Test_Flv_filename(t *testing.T) {
	f := "/mnt/sd/live/hw/2021-09/09-24-085922.flv"
	idx := strings.LastIndex(f, "/")
	tempfile := f[0:idx] + "/temp.flv"

	fmt.Println(tempfile)

}

func Test_file(t *testing.T) {
	filePaths := "D:/work-go/monibuca/resource/live/hk/2021/09/24/143046.flv"

	s := getYearMonthDay(filePaths)
	y := s[0:7]
	t.Log(s)
	t.Log(y)
}

func Test_List_file(t *testing.T) {
	filePaths := "D:/work-go/monibuca/resource"
	month := "2021-09"

	if files, err := tree(filePaths, 0); err == nil {
		var m = make(map[string][]*FlvFileInfo)
		for i := 0; i < len(files); i++ {
			f := files[i]
			day := getYearMonthDay(f.Path) //2021-09
			y := day[0:7]
			if strings.Compare(y, month) == 0 {
				array, _ := m[day]
				array = append(array, f)
				m[day] = array
			}
		}

		j, _ := json.Marshal(m)
		t.Log(j)

		t.Log(m)

	}
}

func Test_StringHex(t *testing.T) {
	a := "4769676162697445746865726E6574302F302F323400"
	bytes, err := hex.DecodeString(a)
	if err != nil {
		t.Error(err)
	}

	t.Logf("%s", string(bytes))
}

func Test_StringHex2(t *testing.T) {
	a := "0123456789"
	bytes := HexEncode(a)

	t.Logf("%s", string(bytes))
	t.Logf(hex.Dump([]byte(a)))
}

func Test_checkfile(t *testing.T) {
	path := "D:\\mnt\\sd\\live\\hw"

	var files []string
	walkFunc := func(itemPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		ext := strings.ToLower(filepath.Ext(itemPath))
		if ext == ".flv" || ext == ".FLV" {
			files = append(files, itemPath)
		}
		return nil
	}

	if err := filepath.Walk(path, walkFunc); err == nil {
		fmt.Printf("%v\n", files)

		delFile := files[0]

		fmt.Println(delFile)

		if err := os.Remove(delFile); err != nil {
			fmt.Errorf("remove file %s error. %s", delFile, err)
		}
	}
}

func Test_for(t *testing.T) {
	for {
		percent, _ := getSdCardUsedPercent2()
		t.Log(percent)
		if percent < 80.00 {
			break
		}
	}
}

var diskspace = 100.00

func getSdCardUsedPercent2() (float64, error) {
	diskspace -= 1.00
	return diskspace, nil
}
