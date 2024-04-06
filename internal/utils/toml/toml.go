package toml

import (
	"github.com/pelletier/go-toml/v2"
	"io"
	"io/ioutil"
	"log"
	"os"
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

func GetApiKeyFromToml() string {
	file, err := os.Open("internal/configs/configs.toml")
	if err != nil {
		log.Println(err)
		return ""
	}
	defer file.Close()
	var cfg config
	b, err := io.ReadAll(file)
	if err != nil {
		log.Println(err)
		return ""
	}
	if err = toml.Unmarshal(b, &cfg); err != nil {
		log.Println(err)
		return ""
	}
	return cfg.Api.Key
}
