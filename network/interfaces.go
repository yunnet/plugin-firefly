package network

import "github.com/yunnet/plugin-firefly/file"

type InterfaceSet struct {
	InterfacesReader

	InterfacesPath string
	Adapters       []*NetworkAdapter
}

func NewInterfaceSet(opts ...file.Option) *InterfaceSet {
	fnConfig := file.MakeConfig(
		file.Defaults{"path": "/etc/network/interfaces"},
		opts,
	)
	path := fnConfig.GetString("path")

	return &InterfaceSet{
		InterfacesPath: path,
	}
}

func Path(path string) file.Option {
	return file.MakeOption("path", path)
}
