package discgo

import (
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"go.uber.org/zap"
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

func InteractionCreateHandler(ses *session.Session, logger *zap.Logger) func(*gateway.InteractionCreateEvent) {
	return func(e *gateway.InteractionCreateEvent) {
		switch data := e.Data.(type) {
		case *discord.CommandInteraction:
			handleCommand(data, logger)
		}

		resp := api.InteractionResponse{
			Type: api.MessageInteractionWithSource,
			Data: &api.InteractionResponseData{Content: option.NewNullableString("test")},
		}

		if err := ses.RespondInteraction(e.ID, e.Token, resp); err != nil {
			logger.Debug("unable to send interaction callback", zap.Error(err))
			return
		}
		if err := ses.DeleteInteractionResponse(e.AppID, e.Token); err != nil {
			logger.Debug("unable to delete interaction response", zap.Error(err))
		}
	}
}

func handleCommand(data *discord.CommandInteraction, logger *zap.Logger) {
	switch data.Name {
	case InsulCommand:
		return
	}
}
