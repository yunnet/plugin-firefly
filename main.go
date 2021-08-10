package firefly

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	. "github.com/Monibuca/engine/v3"
	. "github.com/Monibuca/utils/v3"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	result "github.com/yunnet/plugin-firefly/web"

	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

var config struct{
	Path string
	Username string
	Password string
}

const C_JSON_FILE = "firefly.json"
const C_NETWORK_FILE = "/etc/network/interfaces"

const C_SALT = "firefly"

func init()  {
	InstallPlugin(&PluginConfig{
		Name:   "Firefly",
		Config: &config,
		Run:    run,
	})
}

func run() {
	os.MkdirAll(config.Path, 0755)

	http.HandleFunc("/api/firefly/login", getLoginHandler)

	http.HandleFunc("/api/firefly/config/tcp", getConfigTcpHandler)

	http.HandleFunc("/api/firefly/config", getConfigHandler)
	http.HandleFunc("/api/firefly/config/edit", editConfigHandler)

}

func getLoginHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	requestUser := r.URL.Query().Get("username")
	if requestUser == "" {
		res := result.Err.WithMsg("用户名不能为空")
		w.Write(res.ToRaw())
		return
	}
	requestPassword := r.URL.Query().Get("password")
	if requestPassword == "" {
		res := result.Err.WithMsg("密码不能为空")
		w.Write(res.ToRaw())
		return
	}

	user := config.Username
	if user != requestUser{
		res := result.Err.WithMsg("用户名或密码错误,请重新输入")
		w.Write(res.ToRaw())
		return
	}

	m5 := md5.New()
	m5.Write([]byte(requestPassword + C_SALT))
	password := hex.EncodeToString(m5.Sum(nil))
	if config.Password != password{
		res := result.Err.WithMsg("用户名或密码错误,请重新输入")
		w.Write(res.ToRaw())
		return
	}

	res := result.OK.WithMsg("登陆成功")
	w.Write(res.ToRaw())
}

func getConfigTcpHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	content, err := readFile(C_NETWORK_FILE)
	if nil != err {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.ToRaw())
		return
	}

	res := result.OK.WithData(content)
	w.Write(res.ToRaw())
}

func getConfigHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	if node := r.URL.Query().Get("node"); node != "" {
		filePath := filepath.Join(config.Path, C_JSON_FILE)
		content, err := readFile(filePath)
		if nil != err {
			res := result.Err.WithMsg(err.Error())
			w.Write(res.ToRaw())
			return
		}
		root := gjson.Parse(content)
		nodeJson := root.Get(node)
		res := result.OK.WithData(nodeJson.Raw)
		w.Write(res.ToRaw())
	}else {
		res := result.Err.WithMsg("no such node")
		w.Write(res.ToRaw())
	}
}

func readFile(filePath string) (content string, err error) {
	result, err := ioutil.ReadFile(filePath)
	if nil != err {
		return "", err
	}
	return string(result), nil
}

func editConfigHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil{
		res := result.Err.WithMsg(err.Error())
		w.Write(res.ToRaw())
		return
	}

	defer r.Body.Close()

	CORS(w, r)
	request, err := ioutil.ReadAll(r.Body)
	if err != nil {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.ToRaw())
		return
	}
	requestJson := gjson.Parse(string(request))

	nodePath := requestJson.Get("node")
	if !nodePath.Exists(){
		res := result.Err.WithMsg("param node not exists")
		w.Write(res.ToRaw())
		return
	}
	nodeData := requestJson.Get("data")
	if !nodePath.Exists(){
		res := result.Err.WithMsg("param node data not exists")
		w.Write(res.ToRaw())
		return
	}

	filePath := filepath.Join(config.Path, C_JSON_FILE)
	content, err := readFile(filePath)

	rootJson := gjson.Parse(content)
	node := rootJson.Get(nodePath.String())
	if !node.Exists(){
		res := result.Err.WithMsg("find node not exists")
		w.Write(res.ToRaw())
		return
	}

	if nodePath.String() == "network.tcp"{
		settingIpAddr(nodeData.Raw)
	}

	resultJson, _ := sjson.Set(content, nodePath.String(), nodeData.Value())

	flag := os.O_CREATE | os.O_TRUNC | os.O_WRONLY
	file, err := os.OpenFile(filePath, flag, 0755)
	if err != nil {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.ToRaw())
		return
	}

	file.Write([]byte(resultJson))
	file.Close()

	res := result.OK.WithData("success")
	w.Write(res.ToRaw())
}

func settingIpAddr(content string) error  {
	fmt.Println("recv ip addr：" + content)
	return nil
}


