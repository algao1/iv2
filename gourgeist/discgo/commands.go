package discgo

import (
	"github.com/diamondburned/arikawa/v3/api"
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
		},
		{
			Name:        InsulCommand,
			Description: "Record the estimated insulin intake.",
		},
	}
	return commands
}
