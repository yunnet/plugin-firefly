package web

import "encoding/json"

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func (c Response) WithCode(code int) Response {
	c.Code = code
	return c
}

func (c Response) WithMsg(msg string) Response {
	c.Msg = msg
	return c
}

func (c Response) WithData(data interface{}) Response {
	c.Data = data
	return c
}

func (c *Response) Raw() []byte {
	raw, _ := json.Marshal(c)
	return raw
}

func (c *Response) String() string {
	return string(c.Raw())
}

func response(code int, msg string) *Response {
	return &Response{Code: code, Msg: msg, Data: nil}
}
