package file

import "fmt"

var (
	ErrNotFound = fmt.Errorf("key not found")
)

type Config map[string]interface{}
type Defaults Config

type Option func(Config)

func (c Config) Upsert(key string, value interface{}) {
	c[key] = value
}

// func (c Config) getkey(key string) (value interface{}, err error) {
// 	if val, ok := c[key]; !ok {
// 		return nil, ErrNotFound
// 	} else {
// 		return val, nil
// 	}
// }

func (c Config) Get(key string) interface{} {
	if val, ok := c[key]; !ok {
		return nil
	} else {
		return val
	}
}

func (c Config) GetString(key string) string {
	if val, ok := c[key]; !ok {
		return ""
	} else {
		return val.(string)
	}
}

func MakeConfig(defaults Defaults, options []Option) Config {
	config := Config(defaults)
	for _, fn := range options {
		fn(config)
	}
	return config
}

func MakeOption(key string, value interface{}) Option {
	return func(c Config) {
		c[key] = value
	}
}
