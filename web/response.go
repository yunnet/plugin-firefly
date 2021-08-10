package web

import "encoding/json"

type Response struct {
	Code int			`json:"code"`
	Msg string          `json:"msg"`
	Data interface{}    `json:"data"`
}

func (c *Response)WithMsg(message string) Response {
	return Response{c.Code, message, c.Data}
}

func (c *Response)WithData(data interface{}) Response {
	return Response{c.Code, c.Msg, data}
}

func (c *Response) ToRaw() []byte {
	s := &struct {
		Code int         `json:"code"`
		Msg  string      `json:"msg"`
		Data interface{} `json:"data"`
	}{
		Code: c.Code,
		Msg:  c.Msg,
		Data: c.Data,
	}
	raw, _ := json.Marshal(s)
	return raw
}

func (c *Response) ToString() string {
	return string(c.ToRaw())
}

func response(code int, message string) *Response {
	return &Response{code, message, nil}
}


