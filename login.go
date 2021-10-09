package firefly

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/yunnet/plugin-firefly/jwt"
	result "github.com/yunnet/plugin-firefly/web"
	"log"
	"net/http"
	"os/exec"
	"strings"

	. "github.com/Monibuca/utils/v3"
)

func refreshHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		res := result.Err.WithMsg("Sorry, only GET methods are supported.")
		w.Write(res.Raw())
		return
	}

	tokenString := r.Header.Get("token")
	newTokenString, err := jwt.RefreshToken(tokenString, config.Timeout)
	if err != nil {
		log.Println(err.Error())

		res := result.ErrInvalidToken
		w.Write(res.Raw())
		return
	}

	res := result.OK.WithData(newTokenString)
	w.Write(res.Raw())
}

func CheckLogin(w http.ResponseWriter, r *http.Request) bool {
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

func rebootHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	if r.Method != "GET" {
		res := result.Err.WithMsg("Sorry, only GET methods are supported.")
		w.Write(res.Raw())
		return
	}

	if r.URL.Path != ApiFireflyReboot {
		NotFoundHandler(w, r)
		return
	}
	if isOk := CheckLogin(w, r); !isOk {
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
		res := result.Err.WithMsg("Sorry, only GET methods are supported.")
		w.Write(res.Raw())
		return
	}
	if r.URL.Path != ApiFireflyHi {
		NotFoundHandler(w, r)
		return
	}

	if isOk := CheckLogin(w, r); !isOk {
		return
	}

	res := result.OK
	w.Write(res.Raw())
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, "custom 404")
}

// [Get] /api/firefly/login
func loginHandler(w http.ResponseWriter, r *http.Request) {
	CORS(w, r)
	if r.Method != "GET" {
		res := result.Err.WithMsg("Sorry, only GET methods are supported.")
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

	user := config.Username
	if user != requestUser {
		res := result.Err.WithMsg("用户名或密码错误,请重新输入")
		w.Write(res.Raw())
		return
	}

	m5 := md5.New()
	m5.Write([]byte(requestPassword + C_SALT))
	password := hex.EncodeToString(m5.Sum(nil))
	if strings.Compare(config.Password, password) != 0 {
		res := result.Err.WithMsg("用户名或密码错误,请重新输入")
		w.Write(res.Raw())
		return
	}
	tokenString, _ := jwt.CreateToken(user, config.Timeout)

	res := result.OK.WithData(tokenString)
	w.Write(res.Raw())
}
