package firefly

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func Test_mp4(t *testing.T) {
	filePath := "D:/mnt/sd/record/live/hw/2021-09-28/18-07-25.mp4"
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	duration := GetMP4Duration(file)
	fmt.Println(filepath.Base(filePath), duration)

	t1 := FormatTime(int(duration * 1000))
	fmt.Println(t1)
}
