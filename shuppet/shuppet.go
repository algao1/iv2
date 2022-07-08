package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"iv2/gourgeist/defs"
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

	glucoseTimeout := flag.Int("glucose-timeout", 60, "timeout for high/low glucose alerts")
	noInsulinTimeout := flag.Int("insulin-timeout", 60, "timeout for missing insulin alerts")

	mongoUsername := flag.String("mongo-username", "admin", "mongo username")
	mongoPassword := flag.String("mongo-password", "password", "mongo password")

	flag.Parse()

	cfg := defs.Config{
		Dexcom: defs.DexcomConfig{
			Account:  *dexcomAccount,
			Password: *dexcomPassword,
		},
		Discord: defs.DiscordConfig{
			Token: *discordToken,
			Guild: *discordGuild,
		},
		Mongo: defs.MongoConfig{
			URI:      "mongodb://mongo:27017",
			Username: *mongoUsername,
			Password: *mongoPassword,
		},
		Glucose: defs.GlucoseConfig{
			Low:    *glucoseLow,
			High:   *glucoseHigh,
			Target: *glucoseTarget,
		},
		Alarm: defs.AlarmConfig{
			GlucoseTimeout:   *glucoseTimeout,
			NoInsulinTimeout: *noInsulinTimeout,
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

	envVars := map[string]string{
		"MONGO_USERNAME": *mongoUsername,
		"MONGO_PASSWORD": *mongoPassword,
	}
	envString := ""
	for k, v := range envVars {
		envString += fmt.Sprintln(k + "=" + v)
	}

	err = ioutil.WriteFile("iv2.env", []byte(envString), 0666)
	if err != nil {
		log.Fatal(err)
	}
}
