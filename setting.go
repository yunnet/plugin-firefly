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

const (
	C_JSON_FILE    = "firefly.json"
	C_MNT_SD       = "/mnt/sd"
	C_AUTO_ETH0    = "auto eth0"
	C_IFACE_ETH0   = "iface eth0"
	C_NETWORK_FILE = "/etc/network/interfaces"
	//C_NETWORK_FILE = "/interfaces"
)

var (
	InetList = [4]string{"dhcp", "static", "loopback", "manual"}
)

/**
  [Get] /api/firefly/config/ping
*/
func pingConfigTcpHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	ipAddr := r.URL.Query().Get("ipaddr")
	if ipAddr == "" {
		res := result.Err.WithMsg("ipv4地址不能为空")
		w.Write(res.Raw())
		return
	}
	isOk, err := accessible(ipAddr)
	if isOk {
		res := result.OK.WithMsg("成功")
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
		root := gjson.Parse(content)
		nodeJson := root.Get(node)
		res := result.OK.WithData(nodeJson.Value())
		w.Write(res.Raw())
	} else {
		res := result.Err.WithMsg("no such node")
		w.Write(res.Raw())
	}
}

/**
  [POST] /api/firefly/config/edit
*/
func editConfigHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
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
	requestJson := gjson.Parse(string(request))

	nodePath := requestJson.Get("node")
	if !nodePath.Exists() {
		res := result.Err.WithMsg("param node does not exist.")
		w.Write(res.Raw())
		return
	}
	nodeData := requestJson.Get("data")
	if !nodePath.Exists() {
		res := result.Err.WithMsg("param node data does not exist.")
		w.Write(res.Raw())
		return
	}

	filePath := filepath.Join(config.Path, C_JSON_FILE)
	content, err := readFile(filePath)

	rootJson := gjson.Parse(content)
	node := rootJson.Get(nodePath.String())
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
	if isOk := CheckLogin(w, r); !isOk {
		return
	}

	file := C_NETWORK_FILE
	//file := config.Path + C_NETWORK_FILE
	content, err := readInterfaces(file)
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

	err = updateInterfaces(string(params))
	if err != nil {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.Raw())
	} else {
		res := result.OK.WithMsg("修改成功")
		w.Write(res.Raw())
	}
}

/**
  查看磁盘大小  默认只查看 "/mnt/sd"
  [Get] /api/firefly/storage
*/
func storageHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
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
