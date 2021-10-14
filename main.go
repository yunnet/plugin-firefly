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
	MQTTHost     string
	MQTTUsername string
	MQTTPassword string
	MQTTClientId string
	AlgUrl       string        // 算法源
	TargetUrl    string        // 推送视频流到目标平台地址
	UploadUrl    string        // 上传文件到目标平台地址
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

func initConfig() {
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
	os.MkdirAll(config.Path, 0755)

	initConfig()

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

	//storage
	http.HandleFunc(ApiFireflyConfigStorage, storageHandler)

	http.HandleFunc("/vod/", vodHandler)

	http.HandleFunc("/api/record/list", listHandler)

	http.HandleFunc("/api/record/start", startHandler)

	http.HandleFunc("/api/record/stop", stopHandler)

	http.HandleFunc("/api/record/play", playHandler)

	http.HandleFunc("/api/record/delete", deleteHandler)

	http.HandleFunc("/api/record/download", downloadHandler)

	RunRecord()

	runMQTT(engine.Ctx)
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
