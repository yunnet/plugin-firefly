package firefly

import (
	"encoding/json"
	"errors"
	. "github.com/Monibuca/utils/v3"
	"github.com/bluele/gcache"
	"github.com/jasonlvhit/gocron"
	result "github.com/yunnet/plugin-firefly/web"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

var gc gcache.Cache

type RecFileInfo struct {
	Url       string `json:"url"`
	Size      int64  `json:"size"`
	Timestamp int64  `json:"timestamp"`
	Duration  uint32 `json:"duration"`
}

func (c *RecFileInfo) String() string {
	res, _ := json.Marshal(c)
	return string(res)
}

type FileWr interface {
	io.Reader
	io.Writer
	io.Seeker
	io.Closer
}

func initRecord() {
	gc = gcache.New(100).LRU().Build()

	s := gocron.NewScheduler()
	s.Every(1).Minute().Do(doTask)
	<-s.Start()
}

func doTask() {
	log.Println("at scheduler...")

	for {
		percent, _ := getSdCardUsedPercent()
		log.Printf("disk used %.3f%%\n", percent)

		if percent < C_DISK_SPACE_THRESHOLD {
			break
		}
		freeDisk()
	}
}

func freeDisk() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("internal error: %v", err)
		}
	}()
	var files []string
	walkFunc := func(filePaths string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		ext := strings.ToLower(filepath.Ext(filePaths))
		if ext == ".flv" || ext == ".mp4" {
			_, file := path.Split(filePaths)
			if file[0:1] == "." {
				return nil
			}
			files = append(files, filePaths)
		}
		return nil
	}
	if err := filepath.Walk(config.SavePath, walkFunc); err == nil {
		delFile := files[0]
		log.Println("remove file: " + delFile)

		if err := os.Remove(delFile); err != nil {
			log.Printf("remove file error: %s", err)
		}
	}
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)

	if r.Method != "GET" {
		res := result.Err.WithMsg("Sorry, only GET methods are supported.")
		w.Write(res.Raw())
		return
	}

	if isOk := CheckLogin(w, r); !isOk {
		return
	}

	day := r.URL.Query().Get("date")
	if day == "" {
		res := result.Err.WithMsg("日期不能为空")
		w.Write(res.Raw())
		return
	}

	var files []*RecFileInfo
	walkFunc := func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		ext := strings.ToLower(filepath.Ext(filePath))
		if ext == ".flv" || ext == ".mp4" {
			t := time.Now()
			if f, err := getRecFileInfo(filePath, day); err == nil {
				files = append(files, f)
			}
			spend := time.Since(t).Seconds()
			if spend > 10 {
				log.Printf("spend: %fms", spend)
			}
		}
		return nil
	}

	if err := filepath.Walk(config.SavePath, walkFunc); err == nil {
		if len(files) != 0 {
			res := result.OK.WithData(files)
			w.Write(res.Raw())
		} else {
			res := result.OK.WithData([]interface{}{})
			w.Write(res.Raw())
		}
	}
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)

	if r.Method != "GET" {
		res := result.Err.WithMsg("Sorry, only GET methods are supported.")
		w.Write(res.Raw())
		return
	}

	if isOk := CheckLogin(w, r); !isOk {
		return
	}

	beginStr := r.URL.Query().Get("begin")
	if beginStr == "" {
		res := result.Err.WithMsg("开始日期不能为空")
		w.Write(res.Raw())
		return
	}
	endStr := r.URL.Query().Get("end")
	if endStr == "" {
		res := result.Err.WithMsg("结束日期不能为空")
		w.Write(res.Raw())
		return
	}

	begin, err := StrToDatetime(beginStr)
	if err != nil {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.Raw())
		return
	}

	end, err := StrToDatetime(endStr)
	if err != nil {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.Raw())
		return
	}

	files, _ := getRecords(&begin, &end)
	if len(files) != 0 {
		res := result.OK.WithData(files)
		w.Write(res.Raw())
	} else {
		res := result.OK.WithData([]interface{}{})
		w.Write(res.Raw())
	}
}

func getRecFileRange(dstPath string, begin, end *time.Time) (recFile *RecFileInfo, err error) {
	p := strings.TrimPrefix(dstPath, config.SavePath)
	p = strings.ReplaceAll(p, "\\", "/")

	_, file := path.Split(p)
	if file[0:1] == "." {
		return nil, errors.New("temp file " + file)
	}

	ext := strings.ToLower(path.Ext(file))
	var timestamp time.Time
	if ext == ".flv" {
		timestamp = getFlvTimestamp(p)
	} else if ext == ".mp4" {
		timestamp = getMp4Timestamp(p)
	} else {
		return nil, errors.New("file types do not match")
	}

	if begin.Before(timestamp) && end.After(timestamp) {
		value, err := gc.Get(timestamp)
		if err != nil {
			var f *os.File
			f, err = os.Open(dstPath)
			if err != nil {
				return nil, err
			}
			defer f.Close()

			fileInfo, err := f.Stat()
			if err != nil {
				return nil, err
			}

			if ext == ".flv" {
				recFile = &RecFileInfo{
					Url:       strings.TrimPrefix(p, "/"),
					Size:      fileInfo.Size(),
					Timestamp: timestamp.Unix(),
					Duration:  getDuration(f),
				}
			} else if ext == ".mp4" {
				recFile = &RecFileInfo{
					Url:       strings.TrimPrefix(p, "/"),
					Size:      fileInfo.Size(),
					Timestamp: timestamp.Unix(),
					Duration:  GetMP4Duration(f),
				}
			}
			gc.SetWithExpire(timestamp, recFile, time.Hour*12)
		} else {
			recFile, _ = (value).(*RecFileInfo)
		}
		return recFile, nil
	}

	return nil, errors.New("not found record file")
}

func getRecords(begin, end *time.Time) (files []*RecFileInfo, err error) {
	walkFunc := func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		ext := strings.ToLower(filepath.Ext(filePath))
		if ext == ".flv" || ext == ".mp4" {
			t := time.Now()
			if f, err := getRecFileRange(filePath, begin, end); err == nil && f != nil {
				log.Printf("append file" + f.Url)

				files = append(files, f)
			}
			spend := time.Since(t).Seconds()
			if spend > 10 {
				log.Printf("spend: %fms", spend)
			}
		}
		return nil
	}

	err = filepath.Walk(config.SavePath, walkFunc)
	return
}

func getRecFileInfo(dstPath, findDay string) (recFile *RecFileInfo, err error) {
	p := strings.TrimPrefix(dstPath, config.SavePath)
	p = strings.ReplaceAll(p, "\\", "/")

	if strings.Contains(p, findDay) {
		_, file := path.Split(p)
		if file[0:1] == "." {
			return nil, errors.New("temp file " + file)
		}
		if strings.Contains(p, "alg") {
			return nil, errors.New("alg record file")
		}

		value, err := gc.Get(file)
		if err != nil {
			var f *os.File
			f, err = os.Open(dstPath)
			if err != nil {
				return nil, err
			}
			defer f.Close()

			fileInfo, err := f.Stat()
			if err != nil {
				return nil, err
			}

			ext := strings.ToLower(filepath.Ext(fileInfo.Name()))
			if ext == ".flv" {
				recFile = &RecFileInfo{
					Url:       strings.TrimPrefix(p, "/"),
					Size:      fileInfo.Size(),
					Timestamp: getFlvTimestamp(p).Unix(),
					Duration:  getDuration(f),
				}
			} else if ext == ".mp4" {
				recFile = &RecFileInfo{
					Url:       strings.TrimPrefix(p, "/"),
					Size:      fileInfo.Size(),
					Timestamp: getMp4Timestamp(p).Unix(),
					Duration:  GetMP4Duration(f),
				}
			}
			gc.SetWithExpire(fileInfo.Name(), recFile, time.Hour*12)
		} else {
			recFile, _ = (value).(*RecFileInfo)
		}
		return recFile, nil
	}
	return nil, errors.New("日期不匹配")
}
