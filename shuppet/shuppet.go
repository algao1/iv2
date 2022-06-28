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
			URI: "mongodb://localhost:27017",
		},
		Glucose: gourgeist.GlucoseConfig{
			Low:    4,
			High:   10,
			Target: 6,
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
