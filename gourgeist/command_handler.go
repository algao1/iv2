package gourgeist

import (
	"context"
	"fmt"
	"iv2/gourgeist/discgo"
	"iv2/gourgeist/mongo"
	"iv2/gourgeist/types"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"go.uber.org/zap"
)

const (
	CmdTimeFormat  = "01-02 03:04 PM"
	ExpireDuration = 2 * time.Hour
	LogLimit       = 5
)

type CommandHandler struct {
	Display discgo.Display
	Store   mongo.Store

	Logger   *zap.Logger
	Location *time.Location
}

func (ch *CommandHandler) InteractionCreateHandler() func(*gateway.InteractionCreateEvent) {
	return func(e *gateway.InteractionCreateEvent) {
		var err error
		switch data := e.Data.(type) {
		case *discord.CommandInteraction:
			cmdString := data.Name
			for _, opt := range data.Options {
				cmdString += fmt.Sprintf(" %s %s", opt.Name, opt.String())
			}
			ch.Store.WriteCmdEvent(context.Background(), &types.CommandEvent{
				Time:      time.Now().In(ch.Location),
				CmdString: cmdString,
			})
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
		return ch.handleCarbs(data)
	case discgo.InsulCommand:
		return ch.handleInsul(data)
	default:
		return fmt.Errorf("received unknown command: %s", data.Name)
	}
}

func (ch *CommandHandler) handleCarbs(data *discord.CommandInteraction) error {
	amount, _ := data.Options[0].IntValue()
	ch.Logger.Debug("carbs", zap.Int("amount", int(amount)))

	oldMessage, err := ch.Display.GetMainMessage()
	if err != nil {
		return fmt.Errorf("unable to complete carbs command: %w", err)
	}
	ch.Logger.Debug("old message", zap.Any("embeds", oldMessage.Embeds))

	_, err = ch.Store.WriteCarbs(context.Background(), &types.Carb{
		Time:   time.Now().In(ch.Location),
		Amount: float64(amount),
	})
	if err != nil {
		return fmt.Errorf("unable to save carbs: %w", err)
	}

	err = ch.updateWithEvent(oldMessage)
	if err != nil {
		return fmt.Errorf("unable to complete insul command: %w", err)
	}

	return nil
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
		Time:   time.Now().In(ch.Location),
		Amount: units,
	})
	if err != nil {
		return fmt.Errorf("unable to save insulin: %w", err)
	}

	err = ch.updateWithEvent(oldMessage)
	if err != nil {
		return fmt.Errorf("unable to complete insul command: %w", err)
	}

	return nil
}

func (ch *CommandHandler) updateWithEvent(oldMessage *discord.Message) error {
	desc, err := newDescription(ch.Store, ch.Location)
	if err != nil {
		ch.Logger.Debug("unable to generate new description", zap.Error(err))
	}
	oldMessage.Embeds[0].Description = desc

	return ch.Display.UpdateMainMessage(api.EditMessageData{
		Embeds:      &oldMessage.Embeds,
		Attachments: &[]discord.Attachment{},
	})
}

func newDescription(s mongo.Store, loc *time.Location) (string, error) {
	end := time.Now().In(loc)
	start := end.Add(-ExpireDuration)

	events, err := s.ReadCmdEvents(context.Background(), start, end)
	if err != nil {
		return "", err
	}

	if len(events) == 0 {
		return "", nil
	} else if len(events) > LogLimit {
		events = events[len(events)-LogLimit:]
	}

	desc := "```"
	for _, event := range events {
		desc += fmt.Sprintf("%s %s \n", event.Time.Format(CmdTimeFormat), event.CmdString)
	}
	desc += "```"

	return desc, nil
}
