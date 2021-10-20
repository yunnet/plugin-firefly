package firefly

import (
	"github.com/Monibuca/engine/v3"
	"github.com/Monibuca/utils/v3"
	. "github.com/logrusorgru/aurora"
	"github.com/tidwall/gjson"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var (
	BoxName   = "BoChi"
	SourceUrl = "rtsp://admin:Hw12345678@10.8.72.77:554/LiveMedia/ch1/Media1"
)

var config struct {
	Path         string
	Username     string
	Password     string
	Timeout      time.Duration // 会话超时
	AutoRecord   bool          // 是否自动录制
	SliceStorage bool          // 是否分割文件
	SliceTime    time.Duration // 分割时间
	SavePath     string        // 录制存储路径
	Model        string        // 模式：MO|ZL
	FlvMeta      bool          // 是否补全Flv Metadata
}

func init() {
	engine.InstallPlugin(&engine.PluginConfig{
		Name:   "Firefly",
		Config: &config,
		Run:    run,
		HotConfig: map[string]func(interface{}){
			"AutoRecord": func(v interface{}) {
				config.AutoRecord = v.(bool)
			},
		},
	})
}

func initBoxConfig() {
	filePath := filepath.Join(config.Path, C_JSON_FILE)
	content, err := readFile(filePath)
	if nil != err {
		log.Printf("read firefly.json error " + err.Error())
		return
	}
	SourceUrl = gjson.Get(content, "boxinfo.rtsp").Str
	utils.Print(Green("::::::boxinfo.rtsp: "), BrightBlue(SourceUrl))

	BoxName = gjson.Get(content, "boxinfo.name").Str
	utils.Print(Green("::::::boxinfo.name: "), BrightBlue(BoxName))
}

func run() {
	err := os.MkdirAll(config.Path, 0755)
	if err != nil {
		log.Printf("mkdir %s error: %s", config.Path, err.Error())
		return
	}

	initBoxConfig()

	if config.Model == "ZL" {
		ZLMediaKit()
	}

	//hi
	http.HandleFunc(ApiFireflyHi, hiHandler)

	//登陆
	http.HandleFunc(ApiFireflyLogin, loginHandler)

	//刷新token
	http.HandleFunc(ApiFireflyRefresh, refreshHandler)

	//重启机器
	http.HandleFunc(ApiFireflyReboot, rebootHandler)

	//JSON配置查询
	http.HandleFunc(ApiFireflyConfig, getConfigHandler)
	//JSON配置编辑
	http.HandleFunc(ApiFireflyConfigEdit, editConfigHandler)

	//网络查询
	http.HandleFunc(ApiFireflyConfigTcp, getConfigTcpHandler)
	//网络设置
	http.HandleFunc(ApiFireflyConfigTcpEdit, editConfigTcpHandler)
	//网络ping
	http.HandleFunc(ApiFireflyConfigPing, pingConfigTcpHandler)

	//查看磁盘大小  默认只查看 "/mnt/sd"
	http.HandleFunc(ApiFireflyConfigStorage, storageHandler)

	//查询录制文件列表
	http.HandleFunc("/api/record/list", listHandler)

	//下载录制文件
	http.HandleFunc("/api/record/download", downloadHandler)
}

func ZLMediaKit() {
	var pullStreamUrl = "http://127.0.0.1/index/api/addFFmpegSource?src_url=" + SourceUrl + "&dst_url=rtsp://127.0.0.1/live/hw&timeout_ms=10000&secret=035c73f7-bb6b-4889-a715-d9eb2d1925cc"
	log.Println("pullStreamUrl = " + pullStreamUrl)

	err := httpGet(pullStreamUrl)
	if err != nil {
		log.Printf("pull stream url error. %s \n", err.Error())
	} else {
		log.Println("pull steam ok")
	}
}
