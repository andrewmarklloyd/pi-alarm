package util

import (
	"io/ioutil"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

type stateConfig struct {
	LastKnownStatus string `yaml:"lastKnownStatus"`
}

func ReadState() (stateConfig, error) {
	state := stateConfig{}
	viper.SetConfigName("state")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		return state, err
	}
	err = viper.Unmarshal(&state)
	if err != nil {
		return state, err
	}
	return state, nil
}

func WriteState(state stateConfig) error {
	d, err := yaml.Marshal(&state)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("state.yml", d, 0644)
	if err != nil {
		return err
	}
	return nil
}
