package firefly

import (
	. "github.com/Monibuca/engine/v3"
	. "github.com/Monibuca/utils/v3"
	result "github.com/yunnet/plugin-firefly/web"
	"io"
	"log"
	"net/http"
	"os"
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
	http.HandleFunc("/api/record/download", downloadHandler)

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
	defer func() {
		if err := recover(); err != nil {
			log.Printf("internal error: %v", err)
		}
	}()
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
	begin, err := time.Parse("2006-01-02 15:04:05", beginStr)
	if err != nil {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.Raw())
		return
	}

	endStr := r.URL.Query().Get("end")
	if endStr == "" {
		res := result.Err.WithMsg("结束日期不能为空")
		w.Write(res.Raw())
		return
	}
	end, err := time.Parse("2006-01-02 15:04:05", endStr)
	if err != nil {
		res := result.Err.WithMsg(err.Error())
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
			if f, err := getRecFileRange(filePath, begin, end); err == nil {
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
