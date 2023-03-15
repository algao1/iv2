package main

import (
	"flag"
	"io/ioutil"
	"iv2/gourgeist"
	"iv2/gourgeist/defs"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var configFile string
var realConfigFile defs.Config

func init() {
	flag.StringVar(&configFile, "f", "config.yaml", "config file")
	flag.Parse()
}

func main() {
	logger, _ := zap.NewDevelopment()
	config := defs.Config{Logger: logger}

	file, err := ioutil.ReadFile(configFile)
	if err != nil {
		panic(err)
	}

	if err = yaml.Unmarshal(file, &config); err != nil {
		panic(err)
	}

	gourgeist.Run(config)
}
