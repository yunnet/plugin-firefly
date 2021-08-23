package firefly

import (
	"encoding/json"
	. "github.com/Monibuca/engine/v3"
	. "github.com/Monibuca/utils/v3"
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

	//gocron.Every(1).Day().At("00:00").Do(task)
	if config.DaysStorage {
		s := gocron.NewScheduler()
		s.Every(3).Minute().Do(task)
		//s.Every(1).Day().At("00:00").Do(task)
		<-s.Start()
	}
}

func task() {
	log.Println("at 00:00 task...")

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
	isOk := CheckLogin(w, r)
	if !isOk {
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
	isOk := CheckLogin(w, r)
	if !isOk {
		return
	}

	if streamPath := r.URL.Query().Get("streamPath"); streamPath != "" {
		if err := PublishFlvFile(streamPath); err != nil {
			w.Write([]byte(err.Error()))
		} else {
			w.Write([]byte("success"))
		}
	} else {
		w.Write([]byte("no streamPath"))
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
			w.Write([]byte("success"))
		} else {
			w.Write([]byte("no query stream"))
		}
	} else {
		w.Write([]byte("no such stream"))
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
			w.Write([]byte("success"))
		}
	} else {
		w.Write([]byte("no streamPath"))
	}
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	isOk := CheckLogin(w, r)
	if !isOk {
		return
	}

	if files, err := tree(config.SavePath, 0); err == nil {
		var bytes []byte
		if bytes, err = json.Marshal(files); err == nil {
			w.Write(bytes)
		} else {
			w.Write([]byte("{\"err\":\"" + err.Error() + "\"}"))
		}
	} else {
		w.Write([]byte("{\"err\":\"" + err.Error() + "\"}"))
	}
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	isOk := CheckLogin(w, r)
	if !isOk {
		return
	}

	if streamPath := r.URL.Query().Get("streamPath"); streamPath != "" {
		filePath := filepath.Join(config.SavePath, streamPath+".flv")
		if Exist(filePath) {
			if err := os.Remove(filePath); err != nil {
				w.Write([]byte(err.Error()))
			} else {
				w.Write([]byte("success"))
			}
		} else {
			w.Write([]byte("no such file"))
		}
	} else {
		w.Write([]byte("no streamPath"))
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
