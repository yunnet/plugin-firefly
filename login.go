package firefly

import (
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
	"strings"

	. "github.com/Monibuca/utils/v3"
)

func refreshHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		res := result.Err.WithMsg(ErrorWithGetMethodsSupported)
		w.Write(res.Raw())
		return
	}

	tokenString := r.Header.Get("token")
	newTokenString, err := jwt.RefreshToken(tokenString, Timeout)
	if err != nil {
		log.Println(err.Error())

		res := result.ErrInvalidToken
		w.Write(res.Raw())
		return
	}

	res := result.OK.WithData(newTokenString)
	w.Write(res.Raw())
}

func checkLogin(w http.ResponseWriter, r *http.Request) bool {
	CORS(w, r)
	tokenString := r.Header.Get("token")

	valid, err := jwt.ValidateToken(tokenString)
	if err != nil {
		log.Println(err.Error())

		res := result.ErrUnauthorized
		w.Write(res.Raw())
		return false
	}
	return valid
}

func changePwdHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	if r.Method != "POST" {
		res := result.Err.WithMsg(ErrorWithGetMethodsSupported)
		w.Write(res.Raw())
		return
	}
	if isOk := checkLogin(w, r); !isOk {
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

	oldPass := gjson.Get(params, "password").Str
	newPass := gjson.Get(params, "newpass").Str

	password := jwt.PasswordEncoder(oldPass)
	if strings.Compare(Password, password) != 0 {
		res := result.Err.WithMsg("密码错误,请重新输入")
		w.Write(res.Raw())
		return
	}

	Password = jwt.PasswordEncoder(newPass)

	filePath := filepath.Join(config.Path, C_JSON_FILE)
	content, err := readFile(filePath)

	node := gjson.Get(content, "account")
	if !node.Exists() {
		res := result.Err.WithMsg("account node does not exist.")
		w.Write(res.Raw())
		return
	}

	var infoMap = make(map[string]interface{}, 2)
	infoMap["username"] = Username
	infoMap["password"] = Password
	infoMap["timeout"] = Timeout

	resultJson, _ := sjson.Set(content, "account", infoMap)
	resultJson = jsonFormat(resultJson)

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

func rebootHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	if r.Method != "GET" {
		res := result.Err.WithMsg(ErrorWithGetMethodsSupported)
		w.Write(res.Raw())
		return
	}
	if isOk := checkLogin(w, r); !isOk {
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

// [Get] /api/firefly/config/hi
func hiHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	if r.Method != "GET" {
		res := result.Err.WithMsg(ErrorWithGetMethodsSupported)
		w.Write(res.Raw())
		return
	}
	if isOk := checkLogin(w, r); !isOk {
		return
	}

	res := result.OK
	w.Write(res.Raw())
}

// [Get] /api/firefly/login
func loginHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)

	if r.Method != "GET" {
		res := result.Err.WithMsg(ErrorWithGetMethodsSupported)
		w.Write(res.Raw())
		return
	}

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

	if Username != requestUser {
		res := result.Err.WithMsg("用户名或密码错误,请重新输入")
		w.Write(res.Raw())
		return
	}

	password := jwt.PasswordEncoder(requestPassword)
	if strings.Compare(Password, password) != 0 {
		res := result.Err.WithMsg("用户名或密码错误,请重新输入")
		w.Write(res.Raw())
		return
	}
	tokenString, _ := jwt.CreateToken(Username, Timeout)

	res := result.OK.WithData(tokenString)
	w.Write(res.Raw())
}
