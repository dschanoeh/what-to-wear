package main

import "testing"

func TestLoadConfig(t *testing.T) {
	c := Config{}
	loadConfig("examples/config.yml", &c)
}
