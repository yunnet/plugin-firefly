package firefly

import (
	"bufio"
	"container/list"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-ping/ping"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
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
		return errors.New("IPv4方式不能为空")
	}
	bootproto = strings.ToLower(bootproto)
	if ok := checkInet(bootproto); !ok {
		return errors.New("请正确选择IPv4方式")
	}

	dhcp := false
	if strings.Compare(bootproto, "dhcp") == 0 {
		dhcp = true
	}

	var address, netmask, gateway, nameservers, dns1, dns2 string
	if !dhcp {
		address = gjson.Get(params, "address").Str
		if !checkIp(address) {
			return errors.New("ipv4地址格式不对")
		}

		netmask = gjson.Get(params, "netmask").Str
		if !checkIp(netmask) {
			return errors.New("ipv4子网掩码格式不对")
		}

		gateway = gjson.Get(params, "gateway").Str
		if !checkIp(gateway) {
			return errors.New("ipv4网关地址格式不对")
		}

		dns := gjson.Get(params, "dns")
		if !dns.Exists() {
			return errors.New("dns不能为空")
		}
		dnsList := strings.Split(dns.Str, " ")
		if len(dnsList) > 0 {
			dns1 = dnsList[0]
			if !checkIp(dns1) {
				return errors.New("首选DNS格式不对")
			}
			if len(dnsList) > 1 {
				dns2 = dnsList[1]
				if !checkIp(dns2) {
					return errors.New("备选DNS格式不对")
				}
			}
		} else {
			return errors.New("dns不能为空")
		}
		nameservers = dns1 + " " + dns2
	}

	in, err := os.Open(filePath)
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
	defer func() {
		if err := out.Close(); err == nil {
			log.Println(filePath + " save is ok.")
		} else {
			log.Println("file close error: " + err.Error())
		}
	}()

	if err != nil {
		return err
	}
	for p := l.Front(); p != nil; p = p.Next() {
		line := p.Value.(string)
		log.Println(line)
		out.WriteString(line + "\n")
	}

	return nil
}

//16进制解码
func HexDecode(s string) []byte {
	dst := make([]byte, hex.DecodedLen(len(s))) //申请一个切片, 指明大小. 必须使用hex.DecodedLen
	n, err := hex.Decode(dst, []byte(s))        //进制转换, src->dst
	if err != nil {
		log.Fatal(err)
		return nil
	}
	return dst[:n] //返回0:n的数据.
}

//字符串转为16进制
func HexEncode(s string) []byte {
	dst := make([]byte, hex.EncodedLen(len(s))) //申请一个切片, 指明大小. 必须使用hex.EncodedLen
	n := hex.Encode(dst, []byte(s))             //字节流转化成16进制
	return dst[:n]
}

func UnixTimeFormat(path int64) string {
	return time.Unix(path, 0).Format("2006-01-02")
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
	return fmt.Sprintf("%d天%d小时%d分%d秒%d毫秒", day, hour, minute, second, milliSecond)
}

func httpGet(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	log.Println(string(body))
	return nil
}

//live/hk/2021/09/24/143046.flv
func getFlvTimestamp(path string) int64 {
	return getTimestamp(path, 21, 4, "2006/01/02/150405")
}

//live/hw/2021-09-27/18-07-25.mp4
func getMp4Timestamp(path string) int64 {
	return getTimestamp(path, 23, 4, "2006-01-02/15-04-05")
}

func getTimestamp(path string, start, end int, layout string) int64 {
	s := path[len(path)-start : len(path)-end]
	l, err := time.LoadLocation("Local")
	if err != nil {
		return 0
	}
	tmp, err := time.ParseInLocation(layout, s, l)
	if err != nil {
		return 0
	}
	return tmp.Unix()
}

func getRecFileInfo(dstPath, findDay string) (recFile *RecFileInfo, err error) {
	p := strings.TrimPrefix(dstPath, config.SavePath)
	p = strings.ReplaceAll(p, "\\", "/")

	if strings.Contains(p, findDay) {
		_, file := path.Split(p)
		if file[0:1] == "." {
			return nil, errors.New("temp file " + file)
		}

		value, err := gc.Get(file)
		if err != nil {
			var f *os.File
			f, err = os.Open(dstPath)
			if err != nil {
				return nil, err
			}
			defer f.Close()

			fileInfo, err := f.Stat()
			if err != nil {
				return nil, err
			}

			ext := strings.ToLower(filepath.Ext(fileInfo.Name()))
			if ext == ".flv" {
				recFile = &RecFileInfo{
					Url:       strings.TrimPrefix(p, "/"),
					Size:      fileInfo.Size(),
					Timestamp: getFlvTimestamp(p),
					Duration:  getDuration(f),
				}
			} else if ext == ".mp4" {
				recFile = &RecFileInfo{
					Url:       strings.TrimPrefix(p, "/"),
					Size:      fileInfo.Size(),
					Timestamp: getMp4Timestamp(p),
					Duration:  GetMP4Duration(f),
				}
			}
			gc.SetWithExpire(fileInfo.Name(), recFile, time.Hour*12)
		} else {
			recFile, _ = (value).(*RecFileInfo)
		}
		return recFile, nil
	}
	return nil, errors.New("日期不匹配")
}

func getRecFileRange(dstPath string, begin, end time.Time) (recFile *RecFileInfo, err error) {
	p := strings.TrimPrefix(dstPath, config.SavePath)
	p = strings.ReplaceAll(p, "\\", "/")
	log.Printf("file path = %s", p)

	_, file := path.Split(p)
	if file[0:1] == "." {
		return nil, errors.New("temp file " + file)
	}

	ext := strings.ToLower(path.Ext(file))
	var timestamp int64
	if ext == ".flv" {
		timestamp = getFlvTimestamp(p)
	} else if ext == ".mp4" {
		timestamp = getMp4Timestamp(p)
	} else {
		return nil, errors.New("file ")
	}

	//t := time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
	//b := begin.Format("2006-01-02 15:04:05")
	//e := end.Format("2006-01-02 15:04:05")
	//log.Printf("begin = %s, end = %s, curtime = %s", b, e, t)

	if timestamp >= begin.Unix() && timestamp <= end.Unix() {
		value, err := gc.Get(timestamp)
		if err != nil {
			var f *os.File
			f, err = os.Open(dstPath)
			if err != nil {
				return nil, err
			}
			defer f.Close()

			fileInfo, err := f.Stat()
			if err != nil {
				return nil, err
			}

			if ext == ".flv" {
				recFile = &RecFileInfo{
					Url:       strings.TrimPrefix(p, "/"),
					Size:      fileInfo.Size(),
					Timestamp: timestamp,
					Duration:  getDuration(f),
				}
			} else if ext == ".mp4" {
				recFile = &RecFileInfo{
					Url:       strings.TrimPrefix(p, "/"),
					Size:      fileInfo.Size(),
					Timestamp: timestamp,
					Duration:  GetMP4Duration(f),
				}
			}
			gc.SetWithExpire(timestamp, recFile, time.Hour*12)
		} else {
			recFile, _ = (value).(*RecFileInfo)
		}
		return recFile, nil
	}

	return nil, errors.New("not found record file")
}
