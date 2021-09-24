package firefly

import (
	"bufio"
	"container/list"
	"encoding/json"
	"errors"
	"github.com/go-ping/ping"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
)

func SdCardStat() (*disk.UsageStat, error) {
	return disk.Usage(C_MNT_SD)
}

func getSdCardUsedPercent() (float64, error) {
	sd, err := disk.Usage(C_MNT_SD)
	if err != nil {
		return 0, err
	}
	return sd.UsedPercent, nil
}

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
		return "", nil
	}

	s := bufio.NewScanner(f)
	ready := false
	var ipAddr = make(map[string]interface{}, 5)

	for s.Scan() {
		lines := strings.TrimSpace(s.Text())

		if !ready && (strings.Compare(lines, C_AUTO_ETH0) == 0) {
			ready = true
			continue
		}

		if ready {
			if len(lines) == 0 {
				ready = false
			} else {
				if strings.Contains(lines, C_IFACE_ETH0) {
					ary := strings.Split(lines, " ")
					inet := strings.ToLower(ary[3])
					if strings.Compare(inet, "dhcp") == 0 {
						ready = false
					}
					ipAddr["inet"] = inet
				} else {
					line := strings.SplitN(lines, " ", 2)
					switch line[0] {
					case "address":
						ipAddr["address"] = line[1]
					case "netmask":
						ipAddr["netmask"] = line[1]
					case "gateway":
						ipAddr["gateway"] = line[1]
					case "dns-nameservers":
						ipAddr["dns"] = line[1]
					default:
					}
				}
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

func checkInet(inet string) bool {
	for _, value := range InetList {
		if strings.Compare(value, inet) == 0 {
			return true
		}
	}
	return false
}

func updateInterfaces(params string) error {
	rootJson := gjson.Parse(params)
	dhcp := false

	inet := rootJson.Get("inet").Str
	if len(inet) == 0 {
		return errors.New("IPv4方式不能为空")
	}
	inet = strings.ToLower(inet)

	if ok := checkInet(inet); !ok {
		return errors.New("请正确选择IPv4方式")
	}

	if strings.Compare(inet, "dhcp") == 0 {
		dhcp = true
	}

	var address, netmask, gateway, nameservers string
	if !dhcp {
		address = rootJson.Get("address").Str
		if !checkIp(address) {
			return errors.New("ipv4地址格式不对")
		}

		netmask = rootJson.Get("netmask").Str
		if !checkIp(netmask) {
			return errors.New("ipv4子网掩码格式不对")
		}

		gateway = rootJson.Get("gateway").Str
		if !checkIp(gateway) {
			return errors.New("ipv4网关地址格式不对")
		}

		nameservers = rootJson.Get("dns").Str
		if !checkIp(nameservers) {
			return errors.New("DNS格式不对")
		}
	}

	in, err := os.Open(C_NETWORK_FILE)
	defer in.Close()

	if err != nil {
		return err
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
	log.Printf("ip address [%d] rows affected is Changed", cnt)

	flag := os.O_TRUNC | os.O_CREATE
	out, err := os.OpenFile(C_NETWORK_FILE, flag, 0755)
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
