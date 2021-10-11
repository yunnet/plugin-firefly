//go:build darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd
// +build darwin dragonfly freebsd illumos linux netbsd openbsd

package firefly

import (
	. "github.com/Monibuca/utils/v3"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

func CloseFFmpeg() {
	log.Println(":::::::::::::close FFmpeg.")
	if !Exist(C_PID_FILE) {
		log.Println("gonne.lock file not exists.")
		return
	}
	pid, _ := ioutil.ReadFile(C_PID_FILE)

	cmd := exec.Command("kill", "-9", string(pid))
	log.Println(" => " + cmd.String())

	err := cmd.Start()
	if err != nil {
		log.Println("cmd start", err)
	}
	err = cmd.Wait()
	if err != nil {
		log.Println("cmd wait", err)
	}

	err = os.Remove(C_PID_FILE)
	if err != nil {
		log.Println("cmd remove "+C_PID_FILE, err)
	}
}
