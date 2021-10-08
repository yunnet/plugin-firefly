package firefly

import (
	"bytes"
	"encoding/binary"
	"errors"
	. "github.com/Monibuca/engine/v3"
	. "github.com/Monibuca/utils/v3"
	"github.com/Monibuca/utils/v3/codec"
	"os/exec"
	"strings"

	//amf "github.com/cnotch/ipchub/av/format/amf"
	amf "github.com/zhangpeihao/goamf"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

func getDuration(file FileWr) uint32 {
	_, err := file.Seek(-4, io.SeekEnd)
	if err == nil {
		var tagSize uint32
		if tagSize, err = ReadByteToUint32(file, true); err == nil {
			_, err = file.Seek(-int64(tagSize)-4, io.SeekEnd)
			if err == nil {
				_, timestamp, _, err := codec.ReadFLVTag(file)
				if err == nil {
					return timestamp
				}
			}
		}
	}
	return 0
}

func getSaveFileName(streamPath string) string {
	t := time.Now().Format("2006/01/02/150405")
	return filepath.Join(streamPath, t) + ".flv"
}

func SaveFlv(streamPath string, isAppend bool) error {
	filePath := filepath.Join(config.SavePath, getSaveFileName(streamPath))
	log.Printf(":::::: generate save file name: %s", filePath)

	flag := os.O_CREATE
	if isAppend && !Exist(filePath) {
		isAppend = false
	}
	if isAppend {
		flag = flag | os.O_RDWR | os.O_APPEND
	} else {
		flag = flag | os.O_TRUNC | os.O_WRONLY
	}

	var file FileWr
	var err error
	if ExtraConfig.CreateFileFn == nil {
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}
		file, err = os.OpenFile(filePath, flag, 0755)
		if err != nil {
			return err
		}
	} else {
		file, err = ExtraConfig.CreateFileFn(filePath)
	}

	var offsetTime uint32
	if isAppend {
		offsetTime = getDuration(file)
		file.Seek(0, io.SeekEnd)
	} else {
		_, err = file.Write(codec.FLVHeader)
	}
	if err != nil {
		file.Close()
	}

	// return avformat.WriteFLVTag(file, packet)
	p := Subscriber{
		ID:   filePath,
		Type: "FlvRecord",
	}
	//保存流标签
	recordings.Store(streamPath, &p)

	if err := p.Subscribe(streamPath); err == nil {
		vt, at := p.WaitVideoTrack(), p.WaitAudioTrack()

		var buffer bytes.Buffer
		if _, err := amf.WriteString(&buffer, "onMetaData"); err != nil {
			return err
		}
		metaData := amf.Object{
			"MetaDataCreator": "m7s",
			"hasVideo":        vt != nil,
			"hasAudio":        at != nil,
			"hasMatadata":     true,
			"canSeekToEnd":    false,
			"duration":        0,
			"hasKeyFrames":    0,
			"framerate":       0,
			"videodatarate":   0,
			"filesize":        0,
			"creationdate":    time.Now().Unix(),
		}
		var videoTimes []uint32

		//音频
		p.OnAudio = func(ts uint32, audio *AudioPack) {
			log.Printf("::::::::::::::::::OnAudio::::::::::::::::")

			metaData["videocodecid"] = int(at.CodecID)
			metaData["audiosamplerate"] = at.SoundRate
			metaData["audiosamplesize"] = int(at.SoundSize)
			metaData["stereo"] = at.Channels == 2

			if !isAppend && at.CodecID == 10 { //AAC格式需要发送AAC头
				codec.WriteFLVTag(file, codec.FLV_TAG_TYPE_AUDIO, 0, at.ExtraData)
			}
			codec.WriteFLVTag(file, codec.FLV_TAG_TYPE_AUDIO, ts+offsetTime, audio.Payload)

			p.OnAudio = func(ts uint32, audio *AudioPack) {
				codec.WriteFLVTag(file, codec.FLV_TAG_TYPE_AUDIO, ts+offsetTime, audio.Payload)
			}
		}
		//视频
		p.OnVideo = func(ts uint32, video *VideoPack) {
			log.Printf("::::::::::::::::::OnVideo::::::::::::::::")

			metaData["videocodecid"] = int(vt.CodecID)
			metaData["width"] = vt.SPSInfo.Width
			metaData["height"] = vt.SPSInfo.Height

			if !isAppend {
				codec.WriteFLVTag(file, codec.FLV_TAG_TYPE_VIDEO, 0, vt.ExtraData.Payload)
			}
			codec.WriteFLVTag(file, codec.FLV_TAG_TYPE_VIDEO, ts+offsetTime, video.Payload)

			p.OnVideo = func(ts uint32, video *VideoPack) {
				timestamp := ts + offsetTime
				if video.IDR {
					videoTimes = append(videoTimes, timestamp)
				}
				codec.WriteFLVTag(file, codec.FLV_TAG_TYPE_VIDEO, timestamp, video.Payload)
			}
		}

		go func() {
			p.Play(at, vt)
			log.Printf("timestamp: %v", videoTimes)
			log.Printf("::::::::::::::::::file close::::::::::::::::")
			file.Close()

			if config.FlvMeta {
				go transferFlv(filePath)
			}
		}()
	}
	return err
}

func transferFlv(filename string) {
	log.Printf(":::::::::::::::::transferFlv: %s::::::::::::::::", filename)

	idx := strings.LastIndex(filename, "/")
	tempfile := filename[0:idx] + "/temp.flv"

	nowTime := time.Now()
	if err := exec.Command("yamdi", "-i", filename, "-o", tempfile).Run(); err != nil {
		log.Printf("yamdi -i %s -o %s \n error: %s", filename, tempfile, err.Error())
		return
	}
	endTime := time.Now()

	log.Printf("spend time(s): %f \n", endTime.Sub(nowTime).Seconds())

	err := os.Rename(tempfile, filename)
	if err != nil {
		log.Printf("rename error: %v\n", err)
	}
}

func WriteEcmaArray(w amf.Writer, o amf.Object) (n int, err error) {
	n, err = amf.WriteMarker(w, amf.AMF0_ECMA_ARRAY_MARKER)
	if err != nil {
		return
	}
	length := int32(len(o))
	err = binary.Write(w, binary.BigEndian, &length)
	if err != nil {
		return
	}
	n += 4
	m := 0
	for name, value := range o {
		m, err = amf.WriteObjectName(w, name)
		if err != nil {
			return
		}
		n += m
		m, err = amf.WriteValue(w, value)
		if err != nil {
			return
		}
		n += m
	}
	m, err = amf.WriteObjectEndMarker(w)
	return n + m, err
}

func StopFlv(streamPath string) error {
	if streamPath == "" {
		return errors.New("no streamPath")
	}

	if stream, ok := recordings.Load(streamPath); ok {
		output := stream.(*Subscriber)
		output.Close()
		log.Printf(":::::: stop record file %s is OK.", streamPath)
		return nil
	} else {
		return errors.New("no query stream")
	}
}
