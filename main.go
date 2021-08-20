package firefly

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	. "github.com/Monibuca/engine/v3"
	. "github.com/Monibuca/utils/v3"
	"github.com/go-ping/ping"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/yunnet/plugin-firefly/jwt"
	result "github.com/yunnet/plugin-firefly/web"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var config struct {
	Path     string
	Username string
	Password string
}

const (
	C_JSON_FILE    = "firefly.json"
	C_NETWORK_HEAD = "iface eth0"
	C_NETWORK_FILE = "/etc/network/interfaces"
	C_SALT         = "firefly"
)

func init() {
	InstallPlugin(&PluginConfig{
		Name:   "Firefly",
		Config: &config,
		Run:    run,
	})
}

func run() {
	os.MkdirAll(config.Path, 0755)

	//重启机器
	http.HandleFunc("/api/firefly/reboot", rebootHandler)

	//登陆
	http.HandleFunc("/api/firefly/login", getLoginHandler)

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
}

func pingConfigTcpHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	ipAddr := r.URL.Query().Get("ipaddr")
	if ipAddr == "" {
		res := result.Err.WithMsg("ipv4地址不能为空")
		w.Write(res.Raw())
		return
	}
	isOk, err := Accessible(ipAddr)
	if isOk {
		res := result.OK.WithMsg("成功")
		w.Write(res.Raw())
	} else {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.Raw())
	}
}

func Accessible(ipAddr string) (bool, error) {
	pinger, err := ping.NewPinger(ipAddr)
	if err != nil {
		return false, err
	}
	pinger.Count = 5
	pinger.SetPrivileged(true)

	if err := pinger.Run(); err != nil {
		return false, err
	}

	stats := pinger.Statistics()
	if stats.PacketsRecv >= 1 {
		return true, nil
	}

	return false, errors.New("失败")
}

func checkLogin(w http.ResponseWriter, r *http.Request) bool {
	tokenString := r.Header.Get("token")

	valid, err := jwt.ValidateToken(tokenString)
	if err != nil {
		res := result.ErrUnauthorized
		w.Write(res.Raw())

		log.Println(err.Error())
		return false
	}
	return valid
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
	tokenString, _ := jwt.CreateToken(user, 3600)

	res := result.OK.WithData(tokenString)
	w.Write(res.Raw())
}

func rebootHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	isOk := checkLogin(w, r)
	if !isOk {
		return
	}

	cmd := exec.Command("reboot")
	console, err := cmd.Output()
	if err != nil {
		res := result.Err.WithMsg(err.Error())
		w.Write(res.Raw())
		return
	}

	res := result.OK.WithData(console)
	w.Write(res.Raw())
}

func getConfigHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	isOk := checkLogin(w, r)
	if !isOk {
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

func readFile(filePath string) (content string, err error) {
	res, err := ioutil.ReadFile(filePath)
	if nil != err {
		return "", err
	}
	return string(res), nil
}

func editConfigHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	isOk := checkLogin(w, r)
	if !isOk {
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

func getConfigTcpHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	isOk := checkLogin(w, r)
	if !isOk {
		return
	}

	file := C_NETWORK_FILE
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

func readInterfaces(filePath string) (string, error) {
	f, err := os.Open(filePath)
	defer f.Close()
	if err != nil {
		return "", err
	}

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
	return string(rootJson), nil
}

func CheckIp(ip string) bool {
	addr := strings.Trim(ip, " ")
	regStr := `^(([1-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.)(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){2}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`
	if match, _ := regexp.MatchString(regStr, addr); match {
		return true
	}
	return false
}

func editConfigTcpHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	isOk := checkLogin(w, r)
	if !isOk {
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

func updateInterfaces(params string) error {
	rootJson := gjson.Parse(params)

	address := rootJson.Get("address").Str
	if !CheckIp(address) {
		return errors.New("ipv4地址格式不对")
	}

	netmask := rootJson.Get("netmask").Str
	if !CheckIp(address) {
		return errors.New("ipv4子网掩码格式不对")
	}

	gateway := rootJson.Get("gateway").Str
	if !CheckIp(address) {
		return errors.New("ipv4网关地址格式不对")
	}

	nameservers := rootJson.Get("dns-nameservers").Str
	if !CheckIp(address) {
		return errors.New("DNS格式不对")
	}

	file := C_NETWORK_FILE
	in, err := os.Open(file)
	defer in.Close()
	if err != nil {
		return err
	}

	flag := os.O_RDWR | os.O_CREATE
	out, err := os.OpenFile(file, flag, 0755)
	defer out.Close()
	if err != nil {
		return err
	}

	s := bufio.NewScanner(in)

	ready := false
	for s.Scan() {
		line := s.Text()
		if len(strings.TrimSpace(line)) == 0 {
			if ready {
				ready = false
			}

			_, err = out.WriteString(line + "\n")
			if err != nil {
				return err
			}
			continue
		}

		if strings.Contains(line, C_NETWORK_HEAD) {
			ready = true
			_, err = out.WriteString(line + "\n")
			if err != nil {
				return err
			}
			continue
		}

		var newLine string
		if ready {
			strLine := strings.Split(strings.TrimSpace(line), " ")
			switch strLine[0] {
			case "address":
				newLine = strings.Replace(line, strLine[1], address, -1)
			case "netmask":
				newLine = strings.Replace(line, strLine[1], netmask, -1)
			case "gateway":
				newLine = strings.Replace(line, strLine[1], gateway, -1)
			case "dns-nameservers":
				newLine = strings.Replace(line, strLine[1], nameservers, -1)
			}
			_, err = out.WriteString(newLine + "\n")
			if err != nil {
				return err
			}
		} else {
			_, err = out.WriteString(line + "\n")
			if err != nil {
				return err
			}
		}
	}

	return nil
}
