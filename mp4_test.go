package firefly

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func Test_mp4(t *testing.T) {
	filePath := "D:/yunnet/其它/梦想创未来视频/第二季/2.10梦想创未来-第二季-第十集-古迹.mp4"
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	duration := GetMP4Duration(file)
	fmt.Println(filepath.Base(filePath), duration)

	t1 := FormatTime(int(duration * 1000))
	fmt.Println(t1)
}
