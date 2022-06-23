package discgo

import (
	"iv2/gourgeist/types"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

const (
	CarbsCommand   = "carbs"
	InsulinCommand = "insulin"
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
			Name:        InsulinCommand,
			Description: "Record the estimated insulin intake.",
			Options: discord.CommandOptions{
				&discord.IntegerOption{
					OptionName:  "units",
					Description: "Units of insulin.",
					Min:         option.ZeroInt,
					Required:    true,
				},
				&discord.StringOption{
					OptionName:  "type",
					Description: "Type of insuline (fast, slow).",
					Required:    true,
					Choices: []discord.StringChoice{
						{
							Name:  types.RapidActing.String(),
							Value: types.RapidActing.String(),
						},
						{
							Name:  types.SlowActing.String(),
							Value: types.SlowActing.String(),
						},
					},
				},
			},
		},
	}
	return commands
}
