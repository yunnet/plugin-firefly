package firefly

import (
	"encoding/json"
	"errors"
	. "github.com/Monibuca/engine/v3"
	. "github.com/Monibuca/utils/v3"
	result "github.com/yunnet/plugin-firefly/web"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bluele/gcache"
	"github.com/jasonlvhit/gocron"
)

var (
	recordings sync.Map
	gc         gcache.Cache
)

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

var ExtraConfig struct {
	CreateFileFn     func(filename string) (FileWr, error)
	AutoRecordFilter func(stream string) bool
}

func RunRecord() {
	os.MkdirAll(config.SavePath, 0755)

	gc = gcache.New(100).LRU().Build()

	go AddHook(HOOK_PUBLISH, onPublish)

	http.HandleFunc("/vod/", vodHandler)
	http.HandleFunc("/api/record/list", listHandler)
	http.HandleFunc("/api/record/start", startHandler)
	http.HandleFunc("/api/record/stop", stopHandler)
	http.HandleFunc("/api/record/play", playHandler)
	http.HandleFunc("/api/record/delete", deleteHandler)

	if config.AutoRecord {
		if config.SliceStorage {
			m := config.SliceTime
			if m < 5 {
				m = 5
				log.Printf("record at least %d minutes.", m)
			}
			log.Printf("the current recording is set to %d minutes.", m)

			s := gocron.NewScheduler()
			s.Every(uint64(m)).Minute().Do(doTask)
			<-s.Start()
		}
	}
}

func doTask() {
	log.Printf("at %s task...", time.Now().Format("2006-01-02 15:04:05"))

	checkDisk()

	recordings.Range(func(key, value interface{}) bool {
		streamPath := key.(string)
		StopFlv(streamPath)

		SaveFlv(streamPath, false)
		return true
	})
}

func checkDisk() {
	for {
		percent, _ := getSdCardUsedPercent()
		if percent < C_DISK_SPACE_THRESHOLD {
			break
		}
		freeDisk()
	}
}

func freeDisk() {
	var files []string
	walkFunc := func(itemPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		ext := strings.ToLower(filepath.Ext(itemPath))
		if ext == ".flv" || ext == ".mp4" {
			files = append(files, itemPath)
		}
		return nil
	}
	if err := filepath.Walk(config.SavePath, walkFunc); err == nil {
		delFile := files[0]
		log.Println(delFile)
		if err := os.Remove(delFile); err != nil {
			log.Printf("remove file %s error. %s", delFile, err)
		}
	}
}

func onPublish(p *Stream) {
	if config.AutoRecord || (ExtraConfig.AutoRecordFilter != nil && ExtraConfig.AutoRecordFilter(p.StreamPath)) {
		log.Printf("stream path %s", p.StreamPath)
		SaveFlv(p.StreamPath, false)
	}
}

func vodHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)

	streamPath := r.RequestURI[5:]
	filePath := filepath.Join(config.SavePath, streamPath)
	if file, err := os.Open(filePath); err == nil {
		w.Header().Set("Transfer-Encoding", "chunked")
		w.Header().Set("Content-Type", "video/x-flv")
		io.Copy(w, file)
	} else {
		w.WriteHeader(404)
	}
}

func playHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	if isOk := CheckLogin(w, r); !isOk {
		return
	}

	if streamPath := r.URL.Query().Get("streamPath"); streamPath != "" {
		if err := PublishFlvFile(streamPath); err != nil {
			res := result.Err.WithMsg(err.Error())
			w.Write(res.Raw())
		} else {
			res := result.OK.WithMsg("success")
			w.Write(res.Raw())
		}
	} else {
		res := result.Err.WithMsg("no streamPath")
		w.Write(res.Raw())
	}
}

func stopHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	if isOk := CheckLogin(w, r); !isOk {
		return
	}

	if streamPath := r.URL.Query().Get("streamPath"); streamPath != "" {
		if err := StopFlv(streamPath); err == nil {
			res := result.OK.WithMsg("success")
			w.Write(res.Raw())
		} else {
			res := result.Err.WithMsg("no query stream")
			w.Write(res.Raw())
		}
	} else {
		res := result.Err.WithMsg("no such stream")
		w.Write(res.Raw())
	}
}

func startHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	if isOk := CheckLogin(w, r); !isOk {
		return
	}

	if streamPath := r.URL.Query().Get("streamPath"); streamPath != "" {
		if err := SaveFlv(streamPath, r.URL.Query().Get("append") == "true"); err != nil {
			w.Write([]byte(err.Error()))
		} else {
			res := result.OK.WithMsg("success")
			w.Write(res.Raw())
		}
	} else {
		res := result.Err.WithMsg("no streamPath")
		w.Write(res.Raw())
	}
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	if isOk := CheckLogin(w, r); !isOk {
		return
	}

	if streamPath := r.URL.Query().Get("streamPath"); streamPath != "" {
		filePath := filepath.Join(config.SavePath, streamPath)
		if Exist(filePath) {
			if err := os.Remove(filePath); err != nil {
				res := result.Err.WithMsg(err.Error())
				w.Write(res.Raw())
			} else {
				res := result.OK.WithMsg("success")
				w.Write(res.Raw())
			}
		} else {
			res := result.Err.WithMsg("no such file")
			w.Write(res.Raw())
		}
	} else {
		res := result.Err.WithMsg("no streamPath")
		w.Write(res.Raw())
	}
}

func getRecFileInfo(dstPath, findDay string) (recFile *RecFileInfo, err error) {
	p := strings.TrimPrefix(dstPath, config.SavePath)
	p = strings.ReplaceAll(p, "\\", "/")

	if strings.Contains(p, findDay) {
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

		value, err := gc.Get(fileInfo.Name())
		if err != nil {
			if path.Ext(fileInfo.Name()) == ".flv" {
				recFile = &RecFileInfo{
					Url:       strings.TrimPrefix(p, "/"),
					Size:      fileInfo.Size(),
					Timestamp: getFlvTimestamp(p),
					Duration:  getDuration(f),
				}
			} else if path.Ext(fileInfo.Name()) == ".mp4" {
				recFile = &RecFileInfo{
					Url:       strings.TrimPrefix(p, "/"),
					Size:      fileInfo.Size(),
					Timestamp: getMp4Timestamp(p),
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

//live/hk/2021/09/24/143046.flv
func getFlvTimestamp(path string) int64 {
	return getTimestamp(path, 21, 4, "2006/01/02/150405")
}

//live/hw/2021-09-27/18-07-25.mp4
func getMp4Timestamp(path string) int64 {
	return getTimestamp(path, 23, 4, "2006-01-02/15-04-05")
}

func getTimestamp(path string, start, end int, layout string) int64 {
	s := path[len(path)-start : len(path)-end]
	l, err := time.LoadLocation("Local")
	if err != nil {
		return 0
	}
	tmp, err := time.ParseInLocation(layout, s, l)
	if err != nil {
		return 0
	}
	return tmp.Unix()
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	if isOk := CheckLogin(w, r); !isOk {
		return
	}

	findDay := r.URL.Query().Get("today")
	if findDay == "" {
		res := result.Err.WithMsg("日期不能为空")
		w.Write(res.Raw())
		return
	}

	var files []*RecFileInfo
	walkFunc := func(itemPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		ext := strings.ToLower(filepath.Ext(itemPath))
		if ext == ".flv" || ext == ".mp4" {
			var f *RecFileInfo
			if f, err = getRecFileInfo(itemPath, findDay); err == nil {
				files = append(files, f)
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
