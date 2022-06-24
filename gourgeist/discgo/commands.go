package discgo

import (
	"iv2/gourgeist/types"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

const (
	AddCarbsCmd    = "carbs"
	EditCarbsCmd   = "editcarbs"
	AddInsulinCmd  = "insulin"
	EditInsulinCmd = "editinsulin"
)

func registeredCommands() []api.CreateCommandData {
	commands := []api.CreateCommandData{
		addCarbsCmdData,
		editCarbsCmdData,
		addInsulinCmdData,
		editInsulinCmdData,
	}
	return commands
}

var addCarbsCmdData api.CreateCommandData = api.CreateCommandData{
	Name:        AddCarbsCmd,
	Description: "Record the estimated carbohydrate intake.",
	Options: discord.CommandOptions{
		&discord.IntegerOption{
			OptionName:  "amount",
			Description: "Amount of carbohydrates (grams).",
			Min:         option.ZeroInt,
			Required:    true,
		},
	},
}

var editCarbsCmdData api.CreateCommandData = api.CreateCommandData{
	Name:        EditCarbsCmd,
	Description: "Edit the estimated carbohydrate intake.",
	Options: discord.CommandOptions{
		&discord.StringOption{
			OptionName:  "id",
			Description: "Id of the event to modify or delete.",
			Required:    true,
		},
		&discord.IntegerOption{
			OptionName:  "amount",
			Description: "New amount of carbohydrates (grams). Negative values indicate deletion.",
			Required:    true,
		},
	},
}

var addInsulinCmdData api.CreateCommandData = api.CreateCommandData{
	Name:        AddInsulinCmd,
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
			Description: "Type of insulin (fast, slow).",
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
}

var editInsulinCmdData api.CreateCommandData = api.CreateCommandData{
	Name:        EditInsulinCmd,
	Description: "Edit the estimated insulin intake.",
	Options: discord.CommandOptions{
		&discord.StringOption{
			OptionName:  "id",
			Description: "Id of the event to modify or delete.",
			Required:    true,
		},
		&discord.IntegerOption{
			OptionName:  "units",
			Description: "New units of insulin. Negative values indicate deletion.",
		},
		&discord.StringOption{
			OptionName:  "type",
			Description: "Type of insulin (fast, slow).",
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
}
