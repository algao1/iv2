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

	logger.Debug("loaded config file", zap.Any("config file", config))

	gourgeist.Run(config)
}
