package main

import (
	"flag"
	"io/ioutil"
	"iv2/gourgeist"
	"log"

	"gopkg.in/yaml.v2"
)

func main() {
	dexcomAccount := flag.String("dexcom-account", "", "dexcom account")
	dexcomPassword := flag.String("dexcom-password", "", "dexcom password")

	discordToken := flag.String("discord-token", "", "discord token")
	discordGuild := flag.Int("discord-guild", 0, "discord guild id")

	glucoseLow := flag.Float64("glucose-low", 4, "lower bound for glucose")
	glucoseHigh := flag.Float64("glucose-high", 9, "upper bound for glucose")
	glucoseTarget := flag.Float64("glucose-target", 6, "target glucose")

	flag.Parse()

	cfg := gourgeist.Config{
		Dexcom: gourgeist.DexcomConfig{
			Account:  *dexcomAccount,
			Password: *dexcomPassword,
		},
		Discord: gourgeist.DiscordConfig{
			Token: *discordToken,
			Guild: *discordGuild,
		},
		Mongo: gourgeist.MongoConfig{
			URI: "mongodb://mongo:27017",
		},
		Glucose: gourgeist.GlucoseConfig{
			Low:    *glucoseLow,
			High:   *glucoseHigh,
			Target: *glucoseTarget,
		},
		TrevenantAddr: "trevenant:50051",
		Timezone:      "America/Toronto",
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile("docker-config.yaml", data, 0666)
	if err != nil {
		log.Fatal(err)
	}
}
