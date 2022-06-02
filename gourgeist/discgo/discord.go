package discgo

import (
	"context"
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"go.uber.org/zap"
)

const (
	TimeFormat = "2006-01-02 03:04 PM"

	broadcastChannelName = "iv2"
)

type Discord struct {
	Session  *session.Session
	Logger   *zap.Logger
	Location *time.Location

	gid  discord.GuildID
	chid discord.ChannelID
}

type Display interface {
	GetMainMessage() (*discord.Message, error)
	NewMainMessage(msgData api.SendMessageData) error
	UpdateMainMessage(data api.EditMessageData) error

	// TODO: Eventually separate these out to their own interfaces, too much clutter currently.
	RespondInteraction(id discord.InteractionID, token string, resp api.InteractionResponse) error
	DeleteInteractionResponse(appID discord.AppID, token string) error
}

func New(token string, logger *zap.Logger, loc *time.Location) (*Discord, error) {
	ses := session.NewWithIntents("Bot "+token, gateway.IntentGuildMessages)

	ses.AddIntents(gateway.IntentGuilds)
	ses.AddIntents(gateway.IntentGuildMessages)

	err := ses.Open(context.Background())
	if err != nil {
		return nil, fmt.Errorf("unable to open session: %w", err)
	}

	return &Discord{
		Session:  ses,
		Logger:   logger,
		Location: loc,
	}, nil
}

// TODO: Function signature is overloaded, need addressing.

func (d *Discord) Setup(guildID string, registerCommands bool, handlers ...interface{}) error {
	sf, err := discord.ParseSnowflake(guildID)
	if err != nil {
		return err
	}
	d.gid = discord.GuildID(sf)

	if registerCommands {
		app, err := d.Session.CurrentApplication()
		if err != nil {
			return fmt.Errorf("unable to get current application: %w", err)
		}

		for _, command := range registeredCommands() {
			_, err = d.Session.CreateGuildCommand(app.ID, d.gid, command)
			if err != nil {
				return fmt.Errorf("unable to create guild commands: %w", err)
			}
		}
	}

	for _, handler := range handlers {
		d.Session.AddHandler(handler)
	}

	channels, err := d.Session.Channels(d.gid)
	if err != nil {
		return fmt.Errorf("unable to get channels: %w", err)
	}

	for _, ch := range channels {
		if ch.Type == discord.GuildText && ch.Name == broadcastChannelName {
			d.chid = ch.ID
			return nil
		}
	}

	d.Logger.Debug("creating channel", zap.String("channel name", broadcastChannelName))

	channel, err := d.Session.CreateChannel(d.gid, api.CreateChannelData{
		Name: broadcastChannelName,
		Type: discord.GuildText,
	})
	if err != nil {
		return fmt.Errorf("unable to create channel: %w", err)
	}

	d.Logger.Debug("discord setup complete")

	d.chid = channel.ID
	return nil
}

func (d *Discord) GetMainMessage() (*discord.Message, error) {
	msgs, err := d.Session.Messages(d.chid, 10)
	if err != nil {
		return nil, fmt.Errorf("unable to get messages: %w", err)
	}
	if len(msgs) == 0 {
		d.Logger.Debug("no main message found")
		return nil, nil
	}

	d.Logger.Debug("main message found", zap.Any("msg", msgs[0]))
	return &msgs[0], nil
}

func (d *Discord) deleteOldMessages(chid discord.ChannelID, limit uint) (bool, error) {
	msgs, err := d.Session.Messages(d.chid, limit)
	if err != nil {
		return false, fmt.Errorf("unable to get messages: %w", err)
	}

	for _, msg := range msgs {
		err = d.Session.DeleteMessage(d.chid, msg.ID, api.AuditLogReason("clearing"))
		if err != nil {
			return false, fmt.Errorf("unable to delete message: %w", err)
		}
	}

	msgs, err = d.Session.Messages(d.chid, 1)
	if err != nil {
		return false, fmt.Errorf("unable to get messages: %w", err)
	}
	return len(msgs) == 0, nil
}

func (d *Discord) NewMainMessage(msgData api.SendMessageData) error {
	var clearedAll bool
	var err error

	for !clearedAll {
		clearedAll, err = d.deleteOldMessages(d.chid, 100)
		if err != nil {
			return err
		}
	}

	msg, err := d.Session.SendMessageComplex(d.chid, msgData)
	if err != nil {
		return err
	}
	d.Logger.Debug("created new main message", zap.Any("msg", *msg))

	return nil
}

func (d *Discord) UpdateMainMessage(data api.EditMessageData) error {
	msgs, err := d.Session.Messages(d.chid, 1)
	if err != nil {
		return fmt.Errorf("unable to get messages: %w", err)
	}

	if len(msgs) == 0 {
		msgData := api.SendMessageData{
			Content: data.Content.Val,
			Embeds:  *data.Embeds,
		}
		return d.NewMainMessage(msgData)
	}

	msg, err := d.Session.EditMessageComplex(d.chid, msgs[0].ID, data)
	if err != nil {
		return fmt.Errorf("unable to edit main message: %w", err)
	}
	d.Logger.Debug("updated main message", zap.Any("msg", *msg))

	return nil
}

func (d *Discord) RespondInteraction(id discord.InteractionID, token string, resp api.InteractionResponse) error {
	return d.Session.RespondInteraction(id, token, resp)
}

func (d *Discord) DeleteInteractionResponse(appID discord.AppID, token string) error {
	return d.Session.DeleteInteractionResponse(appID, token)
}
