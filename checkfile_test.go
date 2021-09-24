package firefly

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
