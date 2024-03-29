package main

import (
	"flag"
	"io/ioutil"
	"iv2/gourgeist"
	"iv2/gourgeist/defs"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var (
	configFile string
)

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

	_, err = gourgeist.NewGourgeist(config)
	if err != nil {
		panic(err)
	}

	// Block forever.
	select {}
}
