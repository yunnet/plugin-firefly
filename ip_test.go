package firefly

import (
	"testing"
)

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
