package firefly

import (
	"bufio"
	"bytes"
	"container/list"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	. "github.com/Monibuca/utils/v3"
	"github.com/Monibuca/utils/v3/codec"
	"github.com/go-ping/ping"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/tidwall/gjson"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

var LOC, _ = time.LoadLocation("Asia/Shanghai")

func jsonFormat(data string) string {
	var str bytes.Buffer
	if err := json.Indent(&str, []byte(data), "", "    "); err != nil {
		return ""
	}
	return str.String()
}

func sdCardStat() (*disk.UsageStat, error) {
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
	pinger.Timeout = 10 * time.Second

	if err := pinger.Run(); err != nil {
		return false, err
	}

	stats := pinger.Statistics()
	if stats.PacketsRecv >= 1 {
		return true, nil
	}

	return false, errors.New("error")
}

func readInterfaces(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", nil
	}
	defer f.Close()

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
					bootproto := strings.ToLower(ary[3])
					if strings.Compare(bootproto, "dhcp") == 0 {
						ready = false
					}
					ipAddr["bootproto"] = bootproto
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

func updateInterfaces(params, filePath string) error {
	bootproto := gjson.Get(params, "bootproto").Str
	if len(bootproto) == 0 {
		return errors.New("IPv4??????????????????")
	}
	bootproto = strings.ToLower(bootproto)
	if ok := checkInet(bootproto); !ok {
		return errors.New("???????????????IPv4??????")
	}

	dhcp := false
	if strings.Compare(bootproto, "dhcp") == 0 {
		dhcp = true
	}

	var address, netmask, gateway, nameservers, dns1, dns2 string
	if !dhcp {
		address = gjson.Get(params, "address").Str
		if !checkIp(address) {
			return errors.New("ipv4??????????????????")
		}

		netmask = gjson.Get(params, "netmask").Str
		if !checkIp(netmask) {
			return errors.New("ipv4????????????????????????")
		}

		gateway = gjson.Get(params, "gateway").Str
		if !checkIp(gateway) {
			return errors.New("ipv4????????????????????????")
		}

		dns := gjson.Get(params, "dns")
		if !dns.Exists() {
			return errors.New("dns????????????")
		}
		dnsList := strings.Split(dns.Str, " ")
		if len(dnsList) > 0 {
			dns1 = dnsList[0]
			if !checkIp(dns1) {
				return errors.New("??????DNS????????????")
			}
			if len(dnsList) > 1 {
				dns2 = dnsList[1]
				if !checkIp(dns2) {
					return errors.New("??????DNS????????????")
				}
			}
		} else {
			return errors.New("dns????????????")
		}
		nameservers = dns1 + " " + dns2
	}

	in, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer in.Close()

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
					o := strings.ToLower(strings.TrimSpace(lines[16:]))
					lines = "iface eth0 inet " + bootproto
					l.PushBack(lines)
					if (strings.Compare(o, "dhcp") == 0) && (strings.Compare(o, bootproto) != 0) {
						lines = "address " + address
						l.PushBack(lines)
						lines = "netmask " + netmask
						l.PushBack(lines)
						lines = "gateway " + gateway
						l.PushBack(lines)
						lines = "dns-nameservers " + nameservers
						l.PushBack(lines)
					}
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

	flag := os.O_TRUNC | os.O_CREATE | os.O_WRONLY
	out, err := os.OpenFile(filePath, flag, 0777)
	if err != nil {
		return err
	}
	defer func() {
		if err := out.Close(); err == nil {
			log.Println(filePath + " save is ok.")
		} else {
			log.Println("file close error: " + err.Error())
		}
	}()

	for p := l.Front(); p != nil; p = p.Next() {
		line := p.Value.(string)
		log.Println(line)
		out.WriteString(line + "\n")
	}

	return nil
}

//16????????????
func HexDecode(s string) []byte {
	dst := make([]byte, hex.DecodedLen(len(s))) //??????????????????, ????????????. ????????????hex.DecodedLen
	n, err := hex.Decode(dst, []byte(s))        //????????????, src->dst
	if err != nil {
		log.Fatal(err)
		return nil
	}
	return dst[:n] //??????0:n?????????.
}

//???????????????16??????
func HexEncode(s string) []byte {
	dst := make([]byte, hex.EncodedLen(len(s))) //??????????????????, ????????????. ????????????hex.EncodedLen
	n := hex.Encode(dst, []byte(s))             //??????????????????16??????
	return dst[:n]
}

func UnixTimeFormat(path int64) string {
	return time.Unix(path, 0).Format("2006-01-02")
}

func StrToDatetime(t string) (time.Time, error) {
	return time.ParseInLocation(YYYY_MM_DD_HH_MM_SS, t, LOC)
}

func FormatTime(ms int) string {
	ss := 1000
	mi := ss * 60
	hh := mi * 60
	dd := hh * 24

	day := ms / dd
	hour := (ms - day*dd) / hh
	minute := (ms - day*dd - hour*hh) / mi
	second := (ms - day*dd - hour*hh - minute*mi) / ss
	milliSecond := ms - day*dd - hour*hh - minute*mi - second*ss
	return fmt.Sprintf("%d:%d:%d.%d", hour, minute, second, milliSecond)
}

func FormatTimeStr(ms int) string {
	ss := 1000
	mi := ss * 60
	hh := mi * 60
	dd := hh * 24

	day := ms / dd
	hour := (ms - day*dd) / hh
	minute := (ms - day*dd - hour*hh) / mi
	second := (ms - day*dd - hour*hh - minute*mi) / ss
	milliSecond := ms - day*dd - hour*hh - minute*mi - second*ss
	return fmt.Sprintf("%d???%d??????%d???%d???%d??????", day, hour, minute, second, milliSecond)
}

func httpGet(url string) (string, error) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

//live/hk/2021/09/24/143046.flv
func getFlvTimestamp(path string) time.Time {
	return getTimestamp(path, 21, 4, "2006/01/02/150405")
}

//live/hw/2021-09-27/18-07-25.mp4
func getMp4Timestamp(path string) time.Time {
	return getTimestamp(path, 23, 4, "2006-01-02/15-04-05")
}

func getTimestamp(path string, start, end int, layout string) time.Time {
	s := path[len(path)-start : len(path)-end]
	l, err := time.LoadLocation("Local")
	if err != nil {
		return time.Unix(0, 0)
	}
	tmp, err := time.ParseInLocation(layout, s, l)
	if err != nil {
		return time.Unix(0, 0)
	}
	return tmp
}

func getDuration(file FileWr) uint32 {
	_, err := file.Seek(-4, io.SeekEnd)
	if err == nil {
		var tagSize uint32
		if tagSize, err = ReadByteToUint32(file, true); err == nil {
			_, err = file.Seek(-int64(tagSize)-4, io.SeekEnd)
			if err == nil {
				_, timestamp, _, err := codec.ReadFLVTag(file)
				if err == nil {
					return timestamp
				}
			}
		}
	}
	return 0
}
