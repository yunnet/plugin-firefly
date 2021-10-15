package firefly

import (
	"encoding/json"
	. "github.com/Monibuca/engine/v3"
	. "github.com/Monibuca/utils/v3"
	"github.com/jasonlvhit/gocron"
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
)

var (
	recordings sync.Map
	gc         gcache.Cache
	gCnt       int
	sliceTime  int
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
	go AddHook(HOOK_PUBLISH, onPublish)

	if config.AutoRecord {
		gCnt = 0
		sliceTime = int(config.SliceTime)
		if sliceTime < 5 {
			sliceTime = 5
		}
		log.Printf("record at least %d minutes.", sliceTime)
	}

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

	if config.AutoRecord {
		if config.SliceStorage {
			gCnt++
			if gCnt >= sliceTime {
				log.Printf("the current recording is set to %d minutes.", sliceTime)

				recordings.Range(func(key, value interface{}) bool {
					streamPath := key.(string)
					StopFlv(streamPath)

					SaveFlv(streamPath, false)
					return true
				})

				gCnt = 0
			}
		}
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

func onPublish(p *Stream) {
	if config.AutoRecord || (ExtraConfig.AutoRecordFilter != nil && ExtraConfig.AutoRecordFilter(p.StreamPath)) {
		log.Printf("::::::stream path %s", p.StreamPath)
		SaveFlv(p.StreamPath, false)
	}
}

func vodHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)

	if r.Method != "GET" {
		res := result.Err.WithMsg("Sorry, only GET methods are supported.")
		w.Write(res.Raw())
		return
	}

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

	if r.Method != "GET" {
		res := result.Err.WithMsg("Sorry, only GET methods are supported.")
		w.Write(res.Raw())
		return
	}

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

	if r.Method != "GET" {
		res := result.Err.WithMsg("Sorry, only GET methods are supported.")
		w.Write(res.Raw())
		return
	}

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

	if r.Method != "GET" {
		res := result.Err.WithMsg("Sorry, only GET methods are supported.")
		w.Write(res.Raw())
		return
	}

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

	if r.Method != "GET" {
		res := result.Err.WithMsg("Sorry, only GET methods are supported.")
		w.Write(res.Raw())
		return
	}

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
