package firefly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	. "github.com/Monibuca/utils/v3"
	"github.com/goiiot/libmqtt"
	"github.com/tidwall/gjson"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

/**
功能：
 1、订阅一个主题
 2、接收到开始推流指令后，执行ffmpeg
 3、切换数据源指令：(1)摄像头数据流 (2)算法数据流
 4、接收到停止推流指令后，停止ffmpeg

【开始推流】
指令：{"command": "start"}

【停止推流】
指令：{"command": "stop"}

【切换推流】
指令：{"command": "switch", "enabled": false}

【请求录像列表】
指令：{"command": "record", "begin": "2021-10-11 00:00:00", "end": "2021-10-11 23:59:59"}

【请求上传文件】
指令：{"command": "upload", "file": "live/hw/2021-10-09/15-38-05.mp4"}

*/

var (
	client    libmqtt.Client
	options   []libmqtt.Option
	switchUrl string
	err       error
	topic     string
)

func runMQTT(ctx context.Context) {
	c, cancel := context.WithCancel(ctx)
	defer cancel()

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("收到信号，父context的协程退出,time=", time.Now().Unix())
				destroy()
				return
			default:
				time.Sleep(1 * time.Second)
			}
		}
	}(c)

	topic = "/device/" + config.MQTTClientId

	client, err = libmqtt.NewClient(
		// try MQTT 5.0 and fallback to MQTT 3.1.1
		libmqtt.WithVersion(libmqtt.V311, true),

		// enable keepalive (10s interval) with 20% tolerance
		libmqtt.WithKeepalive(10, 1.2),

		// enable auto reconnect and set backoff strategy
		libmqtt.WithAutoReconnect(true),
		libmqtt.WithBackoffStrategy(time.Second, 5*time.Second, 1.2),

		// use RegexRouter for topic routing if not specified
		// will use TextRouter, which will match full text
		libmqtt.WithRouter(libmqtt.NewRegexRouter()),

		libmqtt.WithConnHandleFunc(connHandler),
		libmqtt.WithNetHandleFunc(netHandler),
		libmqtt.WithSubHandleFunc(subHandler),
		libmqtt.WithUnsubHandleFunc(unSubHandler),
		libmqtt.WithPubHandleFunc(pubHandler),
		libmqtt.WithPersistHandleFunc(persistHandler),
	)

	if err != nil {
		// handle client creation error
		panic("hmm, how could it failed")
	}

	// handle every subscribed message (just for example)
	client.HandleTopic(".*", func(client libmqtt.Client, topic string, qos libmqtt.QosLevel, msg []byte) {
		handleData(client, topic, string(msg))
	})

	options = append(options, libmqtt.WithConnPacket(libmqtt.ConnPacket{
		Username: config.MQTTUsername,
		Password: config.MQTTPassword,
		ClientID: config.MQTTClientId,
	}))

	// connect tcp server
	err = client.ConnectServer(config.MQTTHost, options...)
	if err != nil {
		log.Printf("connect to server failed: %v", err)
	}

	client.Wait()
}

func destroy() {
	CloseFFmpeg()
}

func connHandler(client libmqtt.Client, server string, code byte, err error) {
	if err != nil {
		log.Printf("connect to server [%v] failed: %v", server, err)
		return
	}

	if code != libmqtt.CodeSuccess {
		log.Printf("connect to server [%v] failed with server code [%v]", server, code)
		return
	}

	// connected
	go func() {
		// subscribe to some topics
		client.Subscribe([]*libmqtt.Topic{
			{Name: topic + "/#", Qos: libmqtt.Qos0},
		}...)
	}()
}

func netHandler(client libmqtt.Client, server string, err error) {
	if err != nil {
		log.Printf("error happened to connection to server [%v]: %v", server, err)
	}
}

func persistHandler(client libmqtt.Client, packet libmqtt.Packet, err error) {
	if err != nil {
		log.Printf("session persist error: %v", err)
	}
}

func subHandler(client libmqtt.Client, topics []*libmqtt.Topic, err error) {
	if err != nil {
		for _, t := range topics {
			log.Printf("subscribe to topic [%v] failed: %v", t.Name, err)
		}
	} else {
		for _, t := range topics {
			log.Printf("subscribe to topic [%v] success: %v", t.Name, err)
		}
	}
}

func unSubHandler(client libmqtt.Client, topic []string, err error) {
	if err != nil {
		// handle unsubscribe failure
		for _, t := range topic {
			log.Printf("unsubscribe to topic [%v] failed: %v", t, err)
		}
	} else {
		for _, t := range topic {
			log.Printf("unsubscribe to topic [%v] failed: %v", t, err)
		}
	}
}

func pubHandler(client libmqtt.Client, topic string, err error) {
	if err != nil {
		log.Printf("publish packet to topic [%v] failed: %v", topic, err)
	} else {
		log.Printf("publish packet to topic [%v] success: %v", topic, err)
	}
}

func handleData(client libmqtt.Client, topic, msg string) {
	log.Printf("recv [%v] message: %v", topic, msg)

	commandNode := gjson.Get(msg, "command")

	log.Println(commandNode.Value())

	switch commandNode.String() {
	case "start":
		openFFmpeg(switchUrl)
	case "stop":
		CloseFFmpeg()
	case "switch":
		{
			enabled := gjson.Get(msg, "enabled").Bool()
			switchFFmpeg(enabled)
		}
	case "record":
		getRecordFiles(client, msg)

	case "upload":
		{
			file := gjson.Get(msg, "file").Str
			uploadFile(file)
		}
	default:
		log.Printf("command error %s", commandNode.String())
	}
}

//指令：{"command": "upload", "file": "live/hw/2021-10-09/15-38-05.mp4"}
func uploadFile(file string) {
	filePath := filepath.Join(config.SavePath, file)

	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	//关键的一步操作
	fileWriter, err := bodyWriter.CreateFormFile("uploadfile", filePath)
	if err != nil {
		fmt.Println("error writing to buffer")
		return
	}

	//打开文件句柄操作
	fh, err := os.Open(filePath)
	if err != nil {
		fmt.Println("error opening file")
		return
	}
	defer fh.Close()

	//iocopy
	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		return
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	//POST
	resp, err := http.Post(config.UploadUrl, contentType, bodyBuf)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	fmt.Println(resp.Status)
	fmt.Println(string(respBody))
	return
}

//指令：{"command": "record", "begin": "2021-10-11 00:00:00", "end": "2021-10-11 23:59:59"}
func getRecordFiles(client libmqtt.Client, data string) {
	beginNode := gjson.Get(data, "begin")
	beginStr := beginNode.Str

	endNode := gjson.Get(data, "end")
	endStr := endNode.Str

	var begin, end time.Time
	begin, err = StrToDatetime(beginStr)
	if err != nil {
		payload := "开始日期错误 " + err.Error()
		publish(client, payload)
		return
	}

	end, err = StrToDatetime(endStr)
	if err != nil {
		payload := "开始日期错误 " + err.Error()
		publish(client, payload)
		return
	}

	files, err := getRecords(&begin, &end)
	if err != nil {
		payload := "获取文件列表错误 " + err.Error()
		publish(client, payload)
		return
	}

	res, err := json.Marshal(files)
	if err != nil {
		payload := "获取文件列表错误 " + err.Error()
		publish(client, payload)
		return
	}
	publish(client, string(res))
}

func publish(client libmqtt.Client, payload string) {
	client.Publish([]*libmqtt.PublishPacket{
		{TopicName: topic, Payload: []byte(payload), Qos: libmqtt.Qos0},
	}...)
}

func switchFFmpeg(enabled bool) {
	CloseFFmpeg()

	if enabled {
		switchUrl = config.AlgUrl
	} else {
		switchUrl = SourceUrl
	}
	openFFmpeg(switchUrl)
}

func openFFmpeg(url string) {
	CloseFFmpeg()

	if url == "" {
		log.Println("url is null")
		return
	}
	if Exist(C_PID_FILE) {
		log.Println("ffmpeg already run.")
		return
	}

	cmd := exec.Command("ffmpeg", "-rtsp_transport", "tcp", "-i", url, "-vcodec", "copy", "-acodec", "aac", "-ar", "44100", "-f", "flv", config.TargetUrl)
	log.Println(" => " + cmd.String())
	err := cmd.Start()
	if err != nil {
		log.Println("cmd start", err)
	}

	pid := cmd.Process.Pid
	log.Println("Pid ", pid)

	err = ioutil.WriteFile(C_PID_FILE, []byte(fmt.Sprintf("%d", pid)), 0666)
	if err != nil {
		log.Println("cmd write pid file fail. ", err)
	}

	err = cmd.Wait()
	if err != nil {
		log.Println("cmd wait", err)
	}
}
