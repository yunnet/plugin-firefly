package firefly

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"
)

func Test_timeDuration(t *testing.T) {
	var d time.Duration
	d = 3600
	s := strconv.FormatInt(int64(d), 10)
	t.Log(s)
}

func Test_ip(t *testing.T) {
	ip1 := "192.168.0.111"
	result := checkIp(ip1)
	if result {
		t.Log("pass")
	} else {
		t.Error("fail")
	}
}

func Test_ping(t *testing.T) {
	ipAddr := "10.8.68.18"
	isOk, _ := accessible(ipAddr)
	if isOk {
		t.Log("pass")
	} else {
		t.Error("fail")
	}
}

func Test_Json(t *testing.T) {
	AutoRecord := []bool{true}

	r, _ := json.Marshal(AutoRecord)

	t.Log(r)

	value := make([]interface{}, 0)
	err := json.Unmarshal(r, &value)
	if err != nil {
		t.Log(err.Error())
	} else {
		t.Log(value)
	}

}
