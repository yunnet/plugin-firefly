package firefly

import (
	"context"
	"github.com/Monibuca/engine/v3"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	C_PID_FILE              = "gonne.lock"
	C_SALT                  = "never cast aside and never give up"
	C_JSON_FILE             = "firefly.json"
	C_MNT_SD                = "/mnt/sd"
	C_AUTO_ETH0             = "auto eth0"
	C_IFACE_ETH0            = "iface eth0"
	C_NETWORK_FILE          = "/etc/network/interfaces"
	C_DISK_SPACE_THRESHOLD  = 80.00
	YYYY_MM_DD_HH_MM_SS     = "2006-01-02 15:04:05"
	ApiFireflyHi            = "/api/firefly/hi"
	ApiFireflyLogin         = "/api/firefly/login"
	ApiFireflyRefresh       = "/api/firefly/refresh"
	ApiFireflyReboot        = "/api/firefly/reboot"
	ApiFireflyConfig        = "/api/firefly/config"
	ApiFireflyConfigEdit    = "/api/firefly/config/edit"
	ApiFireflyConfigTcp     = "/api/firefly/config/tcp"
	ApiFireflyConfigTcpEdit = "/api/firefly/config/tcp/edit"
	ApiFireflyConfigPing    = "/api/firefly/config/ping"
	ApiFireflyConfigStorage = "/api/firefly/storage"

	//C_NETWORK_FILE = "/interfaces"
)

var config struct {
	Path     string
	Username string
	Password string
	Timeout  time.Duration // 会话超时

	MQTTHost     string
	MQTTUsername string
	MQTTPassword string
	MQTTClientId string

	SourceUrl string // 拉流源
	TargetUrl string // 推送到目标平台地址

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
	})
}

func ZLMediaKit() {
	var pullStreamUrl = "http://127.0.0.1/index/api/addFFmpegSource?src_url=" + config.SourceUrl + "&dst_url=rtsp://127.0.0.1/live/hw&timeout_ms=10000&secret=035c73f7-bb6b-4889-a715-d9eb2d1925cc"
	log.Println("pullStreamUrl = " + pullStreamUrl)

	err := httpGet(pullStreamUrl)
	if err != nil {
		log.Printf("pull stream url error. %s \n", err.Error())
	} else {
		log.Println("pull steam ok")
	}
}

func run(ctx context.Context) {
	os.MkdirAll(config.Path, 0755)

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

	go runMQTT(ctx)
}
