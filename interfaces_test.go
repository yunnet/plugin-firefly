package firefly

import (
	"bufio"
	"container/list"
	"encoding/json"
	"github.com/tidwall/gjson"
	"os"
	"strings"
	"testing"
)

func Test_read_interfaces(t *testing.T) {
	path := "interfaces"
	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		t.Error(err)
	}

	s := bufio.NewScanner(f)
	ready := false

	var ipAddr = make(map[string]interface{}, 5)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())

		if !ready && (strings.Compare(line, C_AUTO_ETH0) == 0) {
			ready = true
			continue
		}

		if ready {
			if len(line) == 0 {
				ready = false
			} else {
				if strings.Contains(line, C_IFACE_ETH0) {
					ary := strings.Split(line, " ")
					if strings.Compare(ary[3], "dhcp") == 0 {
						ready = false
					}
					ipAddr["inet"] = ary[3]
				} else {
					sline := strings.SplitN(line, " ", 2)
					switch sline[0] {
					case "address":
						ipAddr["address"] = sline[1]
					case "netmask":
						ipAddr["netmask"] = sline[1]
					case "gateway":
						ipAddr["gateway"] = sline[1]
					case "dns-nameservers":
						ipAddr["dns"] = sline[1]
					default:
					}
				}
			}
		}
	}
	rootJson, _ := json.Marshal(ipAddr)
	t.Log(string(rootJson))

}

var (
	params = `{
				"inet": "dhcp"
				}`

	//params = `{
	//			"inet": "static",
	//			"address": "192.168.0.110",
	//			"netmask": "255.255.255.0",
	//			"gateway": "192.168.0.1",
	//			"dns": "144.144.144.144"
	//			}`
)

func Test_update_interfaces(t *testing.T) {
	rootJson := gjson.Parse(params)
	dhcp := false

	inet := rootJson.Get("inet").Str
	if len(inet) == 0 {
		t.Error("inet不能为空")
		return
	}
	inet = strings.ToLower(inet)

	if strings.Compare(inet, "dhcp") == 0 {
		dhcp = true
	}
	var address, netmask, gateway, nameservers string

	if !dhcp {
		address = rootJson.Get("address").Str
		if !checkIp(address) {
			t.Error("ipv4地址格式不对")
			return
		}

		netmask = rootJson.Get("netmask").Str
		if !checkIp(netmask) {
			t.Error("ipv4子网掩码格式不对")
			return
		}

		gateway = rootJson.Get("gateway").Str
		if !checkIp(gateway) {
			t.Error("ipv4网关地址格式不对")
			return
		}

		nameservers = rootJson.Get("dns").Str
		if !checkIp(nameservers) {
			t.Error("DNS格式不对")
			return
		}
	}

	path := "interfaces"
	in, err := os.Open(path)
	defer in.Close()

	if err != nil {
		t.Error(err)
		return
	}

	s := bufio.NewScanner(in)
	l := list.New()

	ready := false
	cnt := 0
	for s.Scan() {
		lines := strings.TrimSpace(s.Text())
		cnt++

		if strings.Compare(lines, C_AUTO_ETH0) == 0 {
			ready = true
			l.PushBack(lines)
			continue
		}

		if ready {
			if len(lines) == 0 {
				ready = false
				l.PushBack(lines)
			} else {
				if strings.Contains(lines, C_IFACE_ETH0) {
					lines = "iface eth0 inet " + inet
					l.PushBack(lines)
				} else {
					if strings.Contains(lines, "address") {
						lines = "address " + address
					} else if strings.Contains(lines, "netmask") {
						lines = "netmask " + netmask
					} else if strings.Contains(lines, "gateway") {
						lines = "gateway " + gateway
					} else if strings.Contains(lines, "dns-nameservers") {
						lines = "dns-nameservers " + nameservers
					}

					if !dhcp {
						l.PushBack(lines)
					}
				}
			}
		} else {
			l.PushBack(lines)
		}
	}
	t.Logf("ip address [%d] rows affected is Changed", cnt)

	for p := l.Front(); p != nil; p = p.Next() {
		line := p.Value.(string)
		t.Log(line + "\n")
	}

	result, _ := json.Marshal(l)
	t.Logf("list: %s", string(result))

	flag := os.O_TRUNC | os.O_CREATE
	out, err := os.OpenFile("file.conf", flag, 0755)
	defer out.Close()
	if err != nil {
		t.Error(err)
		return
	}
	for p := l.Front(); p != nil; p = p.Next() {
		line := p.Value.(string)
		out.WriteString(line + "\n")
	}

}
