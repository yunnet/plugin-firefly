package firefly

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Monibuca/utils/v3/codec"
	amf "github.com/zhangpeihao/goamf"
	"io"
	"os"
	"testing"
)

// 数据名称常量，如元数据
const (
	ScriptOnMetaData = "onMetaData"
)

// MetaData 常见属性名
const (
	MetaDataAudioCodecID    = "audiocodecid"    // Number	音频编解码器 ID
	MetaDataAudioDateRate   = "audiodatarate"   // Number	音频码率，单位 kbps
	MetaDataAudioDelay      = "audiodelay"      // Number	由音频编解码器引入的延时，单位秒
	MetaDataAudioSampleRate = "audiosamplerate" // Number	音频采样率
	MetaDataAudioSampleSize = "audiosamplesize" // Number	音频采样点尺寸
	MetaDataStereo          = "stereo"          // Boolean	音频立体声标志
	MetaDataCanSeekToEnd    = "canSeekToEnd"    // Boolean	指示最后一个视频帧是否是关键帧
	MetaDataCreationDate    = "creationdate"    // String	创建日期与时间
	MetaDataDuration        = "duration"        // Number	文件总时长，单位秒
	MetaDataFileSize        = "filesize"        // Number	文件总长度，单位字节
	MetaDataFrameRate       = "framerate"       // Number	视频帧率
	MetaDataHeight          = "height"          // Number	视频高度，单位像素
	MetaDataVideoCodecID    = "videocodecid"    // Number	视频编解码器 ID
	MetaDataVideoDataRate   = "videodatarate"   // Number	视频码率，单位 kbps
	MetaDataWidth           = "width"           // Number	视频宽度，单位像素
)

func Test_Flv_write(t *testing.T) {
	filepath := "d:/flv/09-17-082109.flv"
	var f FileWr
	var err error
	flag := os.O_CREATE | os.O_TRUNC | os.O_WRONLY
	f, err = os.OpenFile(filepath, flag, 0755)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	var buffer bytes.Buffer
	if _, err := amf.WriteString(&buffer, "onMetaData"); err != nil {
		fmt.Println(err.Error())
		return
	}

	metaData := amf.Object{
		"MetaDataCreator": "m7s",
		"hasVideo":        true,
		"hasAudio":        true,
		"hasMatadata":     true,
		"canSeekToEnd":    false,
		"duration":        0,
		"hasKeyFrames":    0,
		"framerate":       0,
		"videodatarate":   0,
		"filesize":        0,
	}

	if _, err := WriteEcmaArray(&buffer, metaData); err != nil {
		return
	}

	fmt.Println(buffer)

	start, err := f.Seek(13, io.SeekStart)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("start: ", start)

	codec.WriteFLVTag(f, codec.FLV_TAG_TYPE_SCRIPT, 0, buffer.Bytes())

	size := getFileSize(f)
	fmt.Printf("file size = %d\n", size)

	f.Close()

}

func getFileSize(f FileWr) uint64 {
	sum := 0
	buf := make([]byte, 2048)
	for {
		n, err := f.Read(buf)
		sum += n
		if err == io.EOF {
			break
		}
	}
	return uint64(sum)
}

func Test_Duration(t *testing.T) {
	dstPath := "D:/FLV/085839.flv"
	var err error

	var dstF *os.File
	dstF, err = os.Open(dstPath)
	if err != nil {
		return
	}
	defer dstF.Close()

	fileInfo, err := dstF.Stat()
	if err != nil {
		return
	}

	t.Log("file size = ", fileInfo.Size())

	d := getDuration(dstF)
	t.Log("milliSecond = ", d)
	t.Log("time = ", FormatTime(int(d)))
}

func Test_time(t *testing.T) {
	ms := 3551961

	ss := 1000
	mi := ss * 60
	hh := mi * 60
	dd := hh * 24

	day := ms / dd
	hour := (ms - day*dd) / hh
	minute := (ms - day*dd - hour*hh) / mi
	second := (ms - day*dd - hour*hh - minute*mi) / ss
	milliSecond := ms - day*dd - hour*hh - minute*mi - second*ss

	t.Logf("%d天%d小时%d分%d秒%d毫秒", day, hour, minute, second, milliSecond)

	t.Logf("%d:%d:%d.%d", hour, minute, second, milliSecond)
}

func Test_Flv_file_info(t *testing.T) {
	filePaths := "D:/work-go/monibuca/resource/live/hk/2021/09/24/143046.flv"

	f, err := getRecFileInfo(filePaths, "2021-09-29")
	if err != nil {
		t.Log(err)
	}

	j, _ := json.Marshal(f)
	t.Log(string(j))
}