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

	if config.DaysStorage {
		s := gocron.NewScheduler()
		//s.Every(3).Minute().Do(task)
		s.Every(1).Day().At("00:00:01").Do(task)
		<-s.Start()
	}
}

func task() {
	log.Println("at 00:00:01 task...")

	recordings.Range(func(key, value interface{}) bool {
		streamPath := key.(string)
		StopFlv(streamPath)

		SaveFlv(streamPath, false)
		return true
	})
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
	isOk := CheckLogin(w, r)
	if !isOk {
		return
	}

	if streamPath := r.URL.Query().Get("streamPath"); streamPath != "" {
		if err := PublishFlvFile(streamPath); err != nil {
			res := result.Err.WithMsg(err.Error())
			w.Write(res.Raw())
		} else {
			res := result.OK.WithMsg("成功")
			w.Write(res.Raw())
		}
	} else {
		res := result.Err.WithMsg("no streamPath")
		w.Write(res.Raw())
	}
}

func stopHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	isOk := CheckLogin(w, r)
	if !isOk {
		return
	}

	if streamPath := r.URL.Query().Get("streamPath"); streamPath != "" {
		if err := StopFlv(streamPath); err == nil {
			res := result.OK.WithMsg("成功")
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
	isOk := CheckLogin(w, r)
	if !isOk {
		return
	}

	if streamPath := r.URL.Query().Get("streamPath"); streamPath != "" {
		if err := SaveFlv(streamPath, r.URL.Query().Get("append") == "true"); err != nil {
			w.Write([]byte(err.Error()))
		} else {
			res := result.OK.WithMsg("成功")
			w.Write(res.Raw())
		}
	} else {
		res := result.Err.WithMsg("no streamPath")
		w.Write(res.Raw())
	}
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	isOk := CheckLogin(w, r)
	if !isOk {
		return
	}

	if streamPath := r.URL.Query().Get("streamPath"); streamPath != "" {
		filePath := filepath.Join(config.SavePath, streamPath)
		if Exist(filePath) {
			if err := os.Remove(filePath); err != nil {
				res := result.Err.WithMsg(err.Error())
				w.Write(res.Raw())
			} else {
				res := result.OK.WithMsg("成功")
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

func getFileDate(path string) string {
	dir, filename := filepath.Split(path)
	size := len(dir)
	parentDir := string([]byte(dir)[size-8 : size-3])
	days := string([]byte(filename)[:5])
	return parentDir + days
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	isOk := CheckLogin(w, r)
	if !isOk {
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
			curTime := getFileDate(f.Path)
			if strings.Contains(curTime, month) {
				array, _ := m[curTime]
				array = append(array, f)
				m[curTime] = array
			}
		}
		res := result.OK.WithData(m)
		w.Write(res.Raw())
	} else {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.Raw())
	}
}
