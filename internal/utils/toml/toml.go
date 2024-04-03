package toml

import (
	"github.com/pelletier/go-toml/v2"
	"io/ioutil"
)

type config struct {
	Api struct {
		Key string `toml:"key"`
	} `toml:"api"`
}

func PostApiKeyToToml(k string) error {
	cfg := config{
		Api: struct {
			Key string `toml:"key"`
		}(struct{ Key string }{Key: k}),
	}
	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("internal/configs/configs.toml", data, 0644)
	if err != nil {
		return err
	}
	return nil
}
