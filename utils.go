package firefly

import (
	"bufio"
	"container/list"
	"encoding/json"
	"errors"
	"github.com/go-ping/ping"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
)

func readFile(filePath string) (content string, err error) {
	res, err := ioutil.ReadFile(filePath)
	if nil != err {
		return "", err
	}
	return string(res), nil
}

func accessible(ipAddr string) (bool, error) {
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

func checkIp(ip string) bool {
	addr := strings.Trim(ip, " ")
	regStr := `^(([1-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.)(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){2}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`
	if match, _ := regexp.MatchString(regStr, addr); match {
		return true
	}
	return false
}

func updateInterfaces(params string) error {
	rootJson := gjson.Parse(params)

	address := rootJson.Get("address").Str
	if !checkIp(address) {
		return errors.New("ipv4地址格式不对")
	}

	netmask := rootJson.Get("netmask").Str
	if !checkIp(address) {
		return errors.New("ipv4子网掩码格式不对")
	}

	gateway := rootJson.Get("gateway").Str
	if !checkIp(address) {
		return errors.New("ipv4网关地址格式不对")
	}

	nameservers := rootJson.Get("dns-nameservers").Str
	if !checkIp(address) {
		return errors.New("DNS格式不对")
	}

	file := C_NETWORK_FILE
	//file := config.Path + C_NETWORK_FILE
	in, err := os.Open(file)
	defer in.Close()
	if err != nil {
		return err
	}

	s := bufio.NewScanner(in)
	l := list.New()

	ready := false
	cnt := 0
	for s.Scan() {
		line := s.Text()
		cnt++
		if len(strings.TrimSpace(line)) == 0 {
			if ready {
				ready = false
			}
			l.PushBack(line)
			continue
		}

		if strings.Contains(line, C_NETWORK_HEAD) {
			ready = true
			l.PushBack(line)
			continue
		}

		if ready {
			if strings.Contains(line, "address") {
				line = "address " + address
			}

			if strings.Contains(line, "netmask") {
				line = "netmask " + netmask
			}

			if strings.Contains(line, "gateway") {
				line = "gateway " + gateway
			}

			if strings.Contains(line, "dns-nameservers") {
				line = "dns-nameservers " + nameservers
			}
		}

		l.PushBack(line)
	}
	log.Printf("ip address [%d] rows affected is Changed", cnt)

	flag := os.O_RDWR | os.O_CREATE
	out, err := os.OpenFile(file, flag, 0755)
	defer out.Close()
	if err != nil {
		return err
	}
	for p := l.Front(); p != nil; p = p.Next() {
		line := p.Value.(string)
		out.WriteString(line + "\n")
	}

	return nil
}
