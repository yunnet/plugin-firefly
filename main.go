package firefly

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	. "github.com/Monibuca/engine/v3"
	. "github.com/Monibuca/utils/v3"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	result "github.com/yunnet/plugin-firefly/web"
	"io"
	"strings"

	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

var config struct {
	Path     string
	Username string
	Password string
}

const C_JSON_FILE = "firefly.json"
const C_NETWORK_HEAD = "iface eth0"
const C_NETWORK_FILE = "/etc/network/interfaces"

const C_SALT = "firefly"

func init() {
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
	http.HandleFunc("/api/firefly/config/tcp/edit", editConfigTcpHandler)

	http.HandleFunc("/api/firefly/config", getConfigHandler)
	http.HandleFunc("/api/firefly/config/edit", editConfigHandler)

}

func editConfigTcpHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
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
	rootJson := gjson.Parse(string(params))
	address := rootJson.Get("address").Str
	netmask := rootJson.Get("netmask").Str
	gateway := rootJson.Get("gateway").Str
	nameservers := rootJson.Get("dns-nameservers").Str

	fileName := config.Path + C_NETWORK_FILE

	in, err := os.Open(fileName)
	if err != nil {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.Raw())
		return
	}
	defer in.Close()

	//flag := os.O_CREATE | os.O_TRUNC | os.O_WRONLY
	flag := os.O_RDWR | os.O_CREATE
	out, err := os.OpenFile(fileName, flag, 0755)
	if err != nil {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.Raw())
		return
	}
	defer out.Close()

	buf := bufio.NewReader(in)
	ready := false
	for {
		bytes, _, err := buf.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			res := result.Err.WithMsg(err.Error())
			w.Write(res.Raw())
			return
		}

		line := string(bytes)
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			_, err = out.WriteString(line + "\n")
			if err != nil {
				res := result.Err.WithMsg(err.Error())
				w.Write(res.Raw())
				return
			}
			continue
		}

		// Continue if line is empty
		if len(strings.TrimSpace(line)) == 0 {
			if ready {
				ready = false
			}

			_, err = out.WriteString(line + "\n")
			if err != nil {
				res := result.Err.WithMsg(err.Error())
				w.Write(res.Raw())
				return
			}
			continue
		}

		if strings.Contains(line, C_NETWORK_HEAD) {
			ready = true
			_, err = out.WriteString(line + "\n")
			if err != nil {
				res := result.Err.WithMsg(err.Error())
				w.Write(res.Raw())
				return
			}
			continue
		}

		var newLine string
		if ready {
			sline := strings.Split(strings.TrimSpace(line), " ")
			switch sline[0] {
			case "address":
				newLine = strings.Replace(line, sline[1], address, -1)
			case "netmask":
				newLine = strings.Replace(line, sline[1], netmask, -1)
			case "gateway":
				newLine = strings.Replace(line, sline[1], gateway, -1)
			case "dns-nameservers":
				newLine = strings.Replace(line, sline[1], nameservers, -1)
			default:
			}
			_, err = out.WriteString(newLine + "\n")
			if err != nil {
				res := result.Err.WithMsg(err.Error())
				w.Write(res.Raw())
				return
			}
		} else {
			_, err = out.WriteString(line + "\n")
			if err != nil {
				res := result.Err.WithMsg(err.Error())
				w.Write(res.Raw())
				return
			}
		}
	}

	res := result.OK.WithMsg("修改成功")
	w.Write(res.Raw())
}

func getLoginHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	requestUser := r.URL.Query().Get("username")
	if requestUser == "" {
		res := result.Err.WithMsg("用户名不能为空")
		w.Write(res.Raw())
		return
	}
	requestPassword := r.URL.Query().Get("password")
	if requestPassword == "" {
		res := result.Err.WithMsg("密码不能为空")
		w.Write(res.Raw())
		return
	}

	user := config.Username
	if user != requestUser {
		res := result.Err.WithMsg("用户名或密码错误,请重新输入")
		w.Write(res.Raw())
		return
	}

	m5 := md5.New()
	m5.Write([]byte(requestPassword + C_SALT))
	password := hex.EncodeToString(m5.Sum(nil))
	if config.Password != password {
		res := result.Err.WithMsg("用户名或密码错误,请重新输入")
		w.Write(res.Raw())
		return
	}

	res := result.OK.WithMsg("登陆成功")
	w.Write(res.Raw())
}

func getConfigHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
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

func readFile(filePath string) (content string, err error) {
	res, err := ioutil.ReadFile(filePath)
	if nil != err {
		return "", err
	}
	return string(res), nil
}

func editConfigHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.Raw())
		return
	}

	defer r.Body.Close()

	CORS(w, r)
	request, err := ioutil.ReadAll(r.Body)
	if err != nil {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.Raw())
		return
	}
	requestJson := gjson.Parse(string(request))

	nodePath := requestJson.Get("node")
	if !nodePath.Exists() {
		res := result.Err.WithMsg("param node not exists")
		w.Write(res.Raw())
		return
	}
	nodeData := requestJson.Get("data")
	if !nodePath.Exists() {
		res := result.Err.WithMsg("param node data not exists")
		w.Write(res.Raw())
		return
	}

	filePath := filepath.Join(config.Path, C_JSON_FILE)
	content, err := readFile(filePath)

	rootJson := gjson.Parse(content)
	node := rootJson.Get(nodePath.String())
	if !node.Exists() {
		res := result.Err.WithMsg("find node not exists")
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

func getConfigTcpHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	file := config.Path + C_NETWORK_FILE
	content := readInterfaces(file)
	rootJson := gjson.Parse(content)
	res := result.OK.WithData(rootJson.Value())
	w.Write(res.Raw())
}

func readInterfaces(filePath string) string {
	f, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	ready := false

	var ipAddr = make(map[string]interface{}, 4)
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		// Continue if line is empty
		if len(strings.TrimSpace(line)) == 0 {
			if ready {
				break
			}
			continue
		}

		if strings.Contains(line, C_NETWORK_HEAD) {
			ready = true
		}

		if ready {
			sline := strings.Split(strings.TrimSpace(line), " ")
			switch sline[0] {
			case "address":
				ipAddr["address"] = sline[1]
			case "netmask":
				ipAddr["netmask"] = sline[1]
			case "gateway":
				ipAddr["gateway"] = sline[1]
			case "dns-nameservers":
				ipAddr["dns-nameservers"] = sline[1]
			default:
			}
		}
	}
	rootJson, _ := json.Marshal(ipAddr)
	return string(rootJson)
}

func settingIpAddr(content string) error {
	fmt.Println("recv ip addr：" + content)
	return nil
}
