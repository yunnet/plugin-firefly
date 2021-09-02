package firefly

import (
	"github.com/Monibuca/engine/v3"
	"net/http"
	"os"
	"time"
)

var config struct {
	Path        string
	Username    string
	Password    string
	Timeout     time.Duration
	SavePath    string
	AutoRecord  bool
	DaysStorage bool
}

func init() {
	engine.InstallPlugin(&engine.PluginConfig{
		Name:   "Firefly",
		Config: &config,
		Run:    run,
	})
}

func run() {
	os.MkdirAll(config.Path, 0755)

	//登陆
	http.HandleFunc("/api/firefly/login", loginHandler)

	//刷新token
	http.HandleFunc("/api/firefly/refresh", refreshHandler)

	//重启机器
	http.HandleFunc("/api/firefly/reboot", rebootHandler)

	//JSON配置查询
	http.HandleFunc("/api/firefly/config", getConfigHandler)
	//JSON配置编辑
	http.HandleFunc("/api/firefly/config/edit", editConfigHandler)

	//网络查询
	http.HandleFunc("/api/firefly/config/tcp", getConfigTcpHandler)
	//网络设置
	http.HandleFunc("/api/firefly/config/tcp/edit", editConfigTcpHandler)
	//网络ping
	http.HandleFunc("/api/firefly/config/ping", pingConfigTcpHandler)

	RunRecord()
}
