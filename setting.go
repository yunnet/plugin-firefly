package firefly

import (
	. "github.com/Monibuca/utils/v3"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	result "github.com/yunnet/plugin-firefly/web"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

var (
	InetList = [4]string{"dhcp", "static", "loopback", "manual"}
)

/**
  [Get] /api/firefly/config/ping
*/
func pingConfigTcpHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	if r.Method != "GET" {
		res := result.Err.WithMsg("Sorry, only GET methods are supported.")
		w.Write(res.Raw())
		return
	}

	if r.URL.Path != ApiFireflyConfigPing {
		NotFoundHandler(w, r)
		return
	}

	ipAddr := r.URL.Query().Get("ipaddr")
	if ipAddr == "" {
		res := result.Err.WithMsg("ipv4地址不能为空")
		w.Write(res.Raw())
		return
	}
	if !checkIp(ipAddr) {
		res := result.Err.WithMsg("ipv4地址格式不对")
		w.Write(res.Raw())
		return
	}

	isOk, err := accessible(ipAddr)
	if isOk {
		res := result.OK.WithMsg("success")
		w.Write(res.Raw())
	} else {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.Raw())
	}
}

/**
  [Get] /api/firefly/config
*/
func getConfigHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	if r.Method != "GET" {
		res := result.Err.WithMsg("Sorry, only GET methods are supported.")
		w.Write(res.Raw())
		return
	}

	if r.URL.Path != ApiFireflyConfig {
		NotFoundHandler(w, r)
		return
	}
	if isOk := CheckLogin(w, r); !isOk {
		return
	}

	if node := r.URL.Query().Get("node"); node != "" {
		filePath := filepath.Join(config.Path, C_JSON_FILE)
		content, err := readFile(filePath)
		if nil != err {
			res := result.Err.WithMsg(err.Error())
			w.Write(res.Raw())
			return
		}
		nodeJson := gjson.Get(content, node)
		res := result.OK.WithData(nodeJson.Value())
		w.Write(res.Raw())
	} else {
		res := result.Err.WithMsg("no such node")
		w.Write(res.Raw())
	}
}

/**
  [POST] /api/firefly/config/edit
  {"node":"boxinfo","data":{"rtsp":"rtsp://admin:Hw12345678@192.168.0.120:554/LiveMedia/ch1/Media1","name":"haiwei"}}
*/
func editConfigHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	if r.Method != "POST" {
		res := result.Err.WithMsg("Sorry, only POST methods are supported.")
		w.Write(res.Raw())
		return
	}

	if r.URL.Path != ApiFireflyConfigEdit {
		NotFoundHandler(w, r)
		return
	}
	if isOk := CheckLogin(w, r); !isOk {
		return
	}

	if err := r.ParseForm(); err != nil {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.Raw())
		return
	}
	defer r.Body.Close()

	request, err := ioutil.ReadAll(r.Body)
	if err != nil {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.Raw())
		return
	}
	params := string(request)
	if !gjson.Valid(params) {
		res := result.Err.WithMsg("Json format error")
		w.Write(res.Raw())
		return
	}

	//node
	nodePath := gjson.Get(params, "node")
	if !nodePath.Exists() {
		res := result.Err.WithMsg("param node does not exist.")
		w.Write(res.Raw())
		return
	}

	//data
	nodeData := gjson.Get(params, "data")
	if !nodeData.Exists() {
		res := result.Err.WithMsg("param data node does not exist.")
		w.Write(res.Raw())
		return
	}

	filePath := filepath.Join(config.Path, C_JSON_FILE)
	content, err := readFile(filePath)

	node := gjson.Get(content, nodePath.String())
	if !node.Exists() {
		res := result.Err.WithMsg("node does not exist.")
		w.Write(res.Raw())
		return
	}

	resultJson, _ := sjson.Set(content, nodePath.String(), nodeData.Value())

	flag := os.O_CREATE | os.O_TRUNC | os.O_WRONLY
	file, err := os.OpenFile(filePath, flag, 0755)
	if err != nil {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.Raw())
		return
	}

	file.Write([]byte(resultJson))
	file.Close()

	res := result.OK.WithData("success")
	w.Write(res.Raw())
}

/**
  [Get] /api/firefly/config/tcp
*/
func getConfigTcpHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	if r.Method != "GET" {
		res := result.Err.WithMsg("Sorry, only GET methods are supported.")
		w.Write(res.Raw())
		return
	}

	if r.URL.Path != ApiFireflyConfigTcp {
		NotFoundHandler(w, r)
		return
	}

	if isOk := CheckLogin(w, r); !isOk {
		return
	}

	filePath := C_NETWORK_FILE
	//filePath := config.Path + C_NETWORK_FILE
	content, err := readInterfaces(filePath)
	if err != nil {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.Raw())
		return
	}
	rootJson := gjson.Parse(content)
	res := result.OK.WithData(rootJson.Value())
	w.Write(res.Raw())
}

/**
  [POST] /api/firefly/config/tcp/edit
*/
func editConfigTcpHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	if r.Method != "POST" {
		res := result.Err.WithMsg("Sorry, only POST methods are supported.")
		w.Write(res.Raw())
		return
	}

	if r.URL.Path != ApiFireflyConfigTcpEdit {
		NotFoundHandler(w, r)
		return
	}

	if isOk := CheckLogin(w, r); !isOk {
		return
	}

	if err := r.ParseForm(); err != nil {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.Raw())
		return
	}

	params, err := ioutil.ReadAll(r.Body)
	if err != nil {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.Raw())
		return
	}

	filePath := C_NETWORK_FILE
	//filePath := config.Path + C_NETWORK_FILE

	err = updateInterfaces(string(params), filePath)
	if err != nil {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.Raw())
	} else {
		res := result.OK.WithMsg("success")
		w.Write(res.Raw())
	}
}

/**
  查看磁盘大小  默认只查看 "/mnt/sd"
  [Get] /api/firefly/storage
*/
func storageHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	if r.Method != "GET" {
		res := result.Err.WithMsg("Sorry, only GET methods are supported.")
		w.Write(res.Raw())
		return
	}

	if r.URL.Path != ApiFireflyConfigStorage {
		NotFoundHandler(w, r)
		return
	}

	if isOk := CheckLogin(w, r); !isOk {
		return
	}

	if runtime.GOOS == "windows" {
		res := result.Err.WithMsg("windows not support")
		w.Write(res.Raw())
	}

	usageStat, err := SdCardStat()
	if err != nil {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.Raw())
	}
	res := result.OK.WithData(usageStat)
	w.Write(res.Raw())
}
