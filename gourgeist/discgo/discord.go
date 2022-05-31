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
}

type Handler func(*session.Session, *zap.Logger) func(*gateway.InteractionCreateEvent)

// TODO: Not really sure why I made New() and Setup() separate,
//	may change later.

func New(token string, handler func(*session.Session, *zap.Logger) func(*gateway.InteractionCreateEvent), logger *zap.Logger, loc *time.Location) (*Discord, error) {
	ses := session.NewWithIntents("Bot "+token, gateway.IntentGuildMessages)

	ses.AddIntents(gateway.IntentGuilds)
	ses.AddIntents(gateway.IntentGuildMessages)

	err := ses.Open(context.Background())
	if err != nil {
		return nil, fmt.Errorf("unable to open session: %w", err)
	}

	ses.AddHandler(handler(ses, logger))

	return &Discord{
		Session:  ses,
		Logger:   logger,
		Location: loc,
	}, nil
}

func (d *Discord) Setup(guildID string) error {
	sf, err := discord.ParseSnowflake(guildID)
	if err != nil {
		return err
	}
	d.gid = discord.GuildID(sf)

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
		return nil, err
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
		return false, err
	}

	for _, msg := range msgs {
		err = d.Session.DeleteMessage(d.chid, msg.ID, api.AuditLogReason("clearing"))
		if err != nil {
			return false, err
		}
	}

	msgs, err = d.Session.Messages(d.chid, 1)
	if err != nil {
		return false, err
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
