package firefly

import (
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

	"github.com/jasonlvhit/gocron"
)

var recordings sync.Map

type FlvFileInfo struct {
	Path     string
	Size     int64
	Duration uint32
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
	go AddHook(HOOK_PUBLISH, onPublish)
	os.MkdirAll(config.SavePath, 0755)

	http.HandleFunc("/vod/", vodHandler)
	http.HandleFunc("/api/record/list", listHandler)
	http.HandleFunc("/api/record/start", startHandler)
	http.HandleFunc("/api/record/stop", stopHandler)
	http.HandleFunc("/api/record/play", playHandler)
	http.HandleFunc("/api/record/delete", deleteHandler)

	if config.SliceStorage {
		s := gocron.NewScheduler()
		//s.Every(3).Minute().Do(task)
		//s.Every(1).Hour().Do(doTask)
		m := config.SliceTime
		if m < 5 {
			m = 5
			log.Printf("record at least %d minutes.", m)
		}
		log.Printf("the current recording is set to %d minutes.", m)

		s.Every(uint64(m)).Minute().Do(doTask)
		<-s.Start()
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
		if ext == ".flv" || ext == ".FLV" {
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

func tree(dstPath string, level int) (files []*FlvFileInfo, err error) {
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
	if !fileInfo.IsDir() { //如果dstF是文件
		if path.Ext(fileInfo.Name()) == ".flv" {
			p := strings.TrimPrefix(dstPath, config.SavePath)
			p = strings.ReplaceAll(p, "\\", "/")
			files = append(files, &FlvFileInfo{
				Path:     strings.TrimPrefix(p, "/"),
				Size:     fileInfo.Size(),
				Duration: getDuration(dstF),
			})
		}
		return
	} else { //如果dstF是文件夹
		var dir []os.FileInfo
		dir, err = dstF.Readdir(0) //获取文件夹下各个文件或文件夹的fileInfo
		if err != nil {
			return
		}
		for _, fileInfo = range dir {
			var _files []*FlvFileInfo
			_files, err = tree(filepath.Join(dstPath, fileInfo.Name()), level+1)
			if err != nil {
				return
			}
			files = append(files, _files...)
		}
		return
	}
}

func getYearMonthDay(path string) string {
	return strings.ReplaceAll(path[len(path)-21:len(path)-11], "/", "-")
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	if isOk := CheckLogin(w, r); !isOk {
		return
	}

	month := r.URL.Query().Get("month")
	if month == "" {
		res := result.Err.WithMsg("年月不能为空")
		w.Write(res.Raw())
		return
	}

	if files, err := tree(config.SavePath, 0); err == nil {
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
		res := result.OK.WithData(m)
		w.Write(res.Raw())
	} else {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.Raw())
	}
}
