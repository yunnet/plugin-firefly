package firefly

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/yunnet/plugin-firefly/jwt"
	result "github.com/yunnet/plugin-firefly/web"
	"log"
	"net/http"
	"os/exec"

	. "github.com/Monibuca/utils/v3"
)

var (
	C_SALT = "firefly"
)

func refreshHandler(w http.ResponseWriter, r *http.Request) {
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
	isOk := CheckLogin(w, r)
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

func loginHandler(w http.ResponseWriter, r *http.Request) {
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
	tokenString, _ := jwt.CreateToken(user, config.Timeout)

	res := result.OK.WithData(tokenString)
	w.Write(res.Raw())
}
