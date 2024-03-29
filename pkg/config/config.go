package config

import (
	"encoding/json"
	"os"
)

type Step struct {
	Action  string `json:"action"`
	Filter  string `json:"filter"`
	Command string `json:"command"`
}

type Config struct {
	steps []Step
}

func LoadConfig() (Config, error) {
	c := Config{}
	data := os.Getenv("HELMWRAP_CONFIG")
	err := json.Unmarshal([]byte(data), &c.steps)
	return c, err
}

func (c Config) Steps() []Step {
	return c.steps
}

func (c Config) validate() error {
	// TODO
	return nil
}
