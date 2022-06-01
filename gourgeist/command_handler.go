package gourgeist

import (
	"fmt"
	"iv2/gourgeist/discgo"
	"iv2/gourgeist/mongo"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"go.uber.org/zap"
)

type CommandHandler struct {
	Display discgo.Display
	Store   mongo.Store

	Logger *zap.Logger
}

func (ch *CommandHandler) InteractionCreateHandler() func(*gateway.InteractionCreateEvent) {
	return func(e *gateway.InteractionCreateEvent) {
		var err error
		switch data := e.Data.(type) {
		case *discord.CommandInteraction:
			err = ch.handleCommand(data)
		}

		if err != nil {
			// TODO: Add better error handling here.
			return
		}

		resp := api.InteractionResponse{
			Type: api.MessageInteractionWithSource,
			Data: &api.InteractionResponseData{Content: option.NewNullableString("received")},
		}

		if err := ch.Display.RespondInteraction(e.ID, e.Token, resp); err != nil {
			ch.Logger.Debug("unable to send interaction callback", zap.Error(err))
			return
		}
		if err := ch.Display.DeleteInteractionResponse(e.AppID, e.Token); err != nil {
			ch.Logger.Debug("unable to delete interaction response", zap.Error(err))
		}
	}
}

func (ch *CommandHandler) handleCommand(data *discord.CommandInteraction) error {
	ch.Logger.Debug("received command", zap.String("cmd", data.Name))

	switch data.Name {
	case discgo.InsulCommand:
		return nil
	case discgo.CarbsCommand:
		return nil
	default:
		return fmt.Errorf("received unknown command: %s", data.Name)
	}
}
