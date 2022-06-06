package gourgeist

import (
	"context"
	"fmt"
	"iv2/gourgeist/discgo"
	"iv2/gourgeist/mongo"
	"iv2/gourgeist/types"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"go.uber.org/zap"
)

const (
	CmdTimeFormat  = "2006-01-02 03:04 PM -0700"
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

	_, err = ch.Store.WriteCarbs(context.Background(), &types.Carbs{
		Time:   data.ID.Time(),
		Amount: float64(amount),
	})
	if err != nil {
		return fmt.Errorf("unable to save carbs: %w", err)
	}

	err = ch.updateWithEvent(oldMessage, data)
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
		Time:   data.ID.Time(),
		Amount: units,
	})
	if err != nil {
		return fmt.Errorf("unable to save insulin: %w", err)
	}

	err = ch.updateWithEvent(oldMessage, data)
	if err != nil {
		return fmt.Errorf("unable to complete insul command: %w", err)
	}

	return nil
}

func (ch *CommandHandler) updateWithEvent(oldMessage *discord.Message, data *discord.CommandInteraction) error {
	if len(oldMessage.Embeds) > 0 {
		cl := CommandLog{
			Time: time.Now(),
			Log:  data.Name,
		}
		for _, opt := range data.Options {
			cl.Log += fmt.Sprintf(" %s %s", opt.Name, opt.Value.String())
		}

		desc, err := newDesc(oldMessage.Embeds[0].Description, ch.Location, cl)
		if err != nil {
			ch.Logger.Debug("unable to generate new desc", zap.Error(err))
		} else {
			oldMessage.Embeds[0].Description = desc
		}
	}

	return ch.Display.UpdateMainMessage(api.EditMessageData{
		Embeds:      &oldMessage.Embeds,
		Attachments: &[]discord.Attachment{},
	})
}

type CommandLog struct {
	Time time.Time
	Log  string
}

func newDesc(oldDesc string, loc *time.Location, logs ...CommandLog) (string, error) {
	oldLogs, err := descToLogs(oldDesc, loc)
	if err != nil {
		return "", fmt.Errorf("unable to convert desc to logs: %w", err)
	}
	oldLogs = append(oldLogs, logs...)
	if len(oldLogs) > LogLimit {
		oldLogs = oldLogs[len(oldLogs)-LogLimit:]
	}

	newLogs := make([]CommandLog, 0)
	for _, log := range oldLogs {
		if log.Time.Add(ExpireDuration).After(time.Now()) {
			newLogs = append(newLogs, log)
		}
	}
	return logsToDesc(newLogs, loc), nil
}

func logsToDesc(logs []CommandLog, loc *time.Location) string {
	if len(logs) == 0 {
		return ""
	}

	desc := "```"
	for i, log := range logs {
		desc += log.Time.In(loc).Format(CmdTimeFormat) + " | " + log.Log
		if i != len(logs)-1 {
			desc += "\n"
		}
	}
	desc += "```"
	return desc
}

func descToLogs(desc string, loc *time.Location) ([]CommandLog, error) {
	logs := make([]CommandLog, 0)
	if len(desc) == 0 {
		return nil, nil
	}

	desc = strings.ReplaceAll(desc, "`", "")
	lines := strings.Split(desc, "\n")
	for _, line := range lines {
		split := strings.Split(line, "|")
		if len(split) != 2 {
			return nil, fmt.Errorf("invalid format")
		}

		t, err := time.Parse(CmdTimeFormat, strings.TrimSpace(split[0]))
		if err != nil {
			return nil, fmt.Errorf("unable to parse time: %w", err)
		}
		logs = append(logs, CommandLog{Time: t, Log: strings.TrimSpace(split[1])})
	}

	return logs, nil
}
