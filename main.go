package firefly

import (
	. "github.com/Monibuca/engine/v3"
	"os"
)

var config struct {
	Path     string
	Username string
	Password string

	SavePath    string
	AutoRecord  bool
	DaysStorage bool
}

func init() {
	InstallPlugin(&PluginConfig{
		Name:   "Firefly",
		Config: &config,
		Run:    run,
	})
}

func run() {
	os.MkdirAll(config.Path, 0755)

	RunLogin()

	RunSetting()

	RunRecord()
}
