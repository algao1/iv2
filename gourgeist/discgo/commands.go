package discgo

import (
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

const (
	CarbsCommand = "carbs"
	InsulCommand = "insul"
)

func registeredCommands() []api.CreateCommandData {
	commands := []api.CreateCommandData{
		{
			Name:        CarbsCommand,
			Description: "Record the estimated carbohydrate intake.",
			Options: discord.CommandOptions{
				&discord.IntegerOption{
					OptionName:  "amount",
					Description: "Amount of carbohydrates (grams).",
					Min:         option.ZeroInt,
					Required:    true,
				},
			},
		},
		{
			Name:        InsulCommand,
			Description: "Record the estimated insulin intake.",
			Options: discord.CommandOptions{
				&discord.IntegerOption{
					OptionName:  "units",
					Description: "Units of insulin.",
					Min:         option.ZeroInt,
					Required:    true,
				},
			},
		},
	}
	return commands
}
