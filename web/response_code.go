package web

var (
	OK  = response(200, "ok")
	Err = response(500, "")

	ErrUnauthorized = response(10001, "访问未授权")
	ErrInvalidToken = response(10002, "无效token")
)
