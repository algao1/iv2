package gourgeist

import (
	"context"
	"fmt"
	"iv2/gourgeist/discgo"
	"iv2/gourgeist/mongo"
	"iv2/gourgeist/types"

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
			ch.Logger.Debug("unable to handle command", zap.Error(err))
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
	case discgo.CarbsCommand:
		return nil
	case discgo.InsulCommand:
		return ch.handleInsul(data)
	default:
		return fmt.Errorf("received unknown command: %s", data.Name)
	}
}

func (ch *CommandHandler) handleInsul(data *discord.CommandInteraction) error {
	units, _ := data.Options[0].FloatValue()
	ch.Logger.Debug("insulin", zap.Float64("units", units))

	oldMessage, err := ch.Display.GetMainMessage()
	if err != nil {
		return fmt.Errorf("unable to complete insul command: %w", err)
	}
	ch.Logger.Debug("old message", zap.Any("embeds", oldMessage.Embeds))

	_, err = ch.Store.WriteInsulin(context.Background(), &types.Insulin{
		Time:   data.ID.Time(),
		Amount: units,
	})
	if err != nil {
		return fmt.Errorf("unable to save insulin: %w", err)
	}

	err = ch.Display.UpdateMainMessage(api.EditMessageData{
		Embeds:      &oldMessage.Embeds,
		Attachments: &[]discord.Attachment{},
	})
	if err != nil {
		return fmt.Errorf("unable to complete insul command: %w", err)
	}

	return nil
}
