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

	mainChannel = "iv2"
)

type Discord struct {
	Session  *session.Session
	Logger   *zap.Logger
	Location *time.Location

	gid      discord.GuildID
	channels map[string]discord.ChannelID
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

	if err := ses.Open(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to open session: %w", err)
	}

	return &Discord{
		Session:  ses,
		Logger:   logger,
		Location: loc,
	}, nil
}

// TODO: Function signature is overloaded, need addressing.

func (d *Discord) Setup(guildID string, cmds []api.CreateCommandData, channels []string, handlers ...interface{}) error {
	sf, err := discord.ParseSnowflake(guildID)
	if err != nil {
		return err
	}
	d.gid = discord.GuildID(sf)

	app, err := d.Session.CurrentApplication()
	if err != nil {
		return fmt.Errorf("unable to get current application: %w", err)
	}

	commands, err := d.Session.Commands(app.ID)
	if err != nil {
		return fmt.Errorf("unable to fetch commands: %w", err)
	}

	// Delete old commands.
	for _, command := range commands {
		d.Session.DeleteCommand(app.ID, command.ID)
		d.Logger.Info("deleted command",
			zap.Any("command id", command.ID),
			zap.String("command name", command.Name),
		)
	}

	// Create commands.
	for _, cmd := range cmds {
		if _, err = d.Session.CreateGuildCommand(app.ID, d.gid, cmd); err != nil {
			return fmt.Errorf("unable to create guild commands: %w", err)
		}
	}

	for _, handler := range handlers {
		d.Session.AddHandler(handler)
	}

	d.channels = make(map[string]discord.ChannelID)

	// Populate existing channels.
	existChannels, err := d.Session.Channels(d.gid)
	if err != nil {
		return fmt.Errorf("unable to get channels: %w", err)
	}
	for _, ch := range existChannels {
		d.channels[ch.Name] = ch.ID
	}

	// Ensure main channel is created.
	channels = append(channels, mainChannel)

	for _, chName := range channels {
		if _, ok := d.channels[chName]; !ok {
			d.Logger.Debug("creating channel", zap.String("channel name", chName))
			ch, err := d.Session.CreateChannel(d.gid, api.CreateChannelData{
				Name: chName,
				Type: discord.GuildText,
			})
			if err != nil {
				return fmt.Errorf("unable to create channel %s: %w", chName, err)
			}
			d.channels[chName] = ch.ID
		}
	}

	d.Logger.Debug("discord setup complete")
	return nil
}

func (d *Discord) GetMainMessage() (*discord.Message, error) {
	msgs, err := d.Session.Messages(d.channels[mainChannel], 10)
	if err != nil {
		return nil, fmt.Errorf("unable to get messages: %w", err)
	}
	if len(msgs) == 0 {
		return nil, fmt.Errorf("no main message found")
	}
	return &msgs[0], nil
}

func (d *Discord) deleteOldMessages(chid discord.ChannelID, limit uint) (bool, error) {
	msgs, err := d.Session.Messages(d.channels[mainChannel], limit)
	if err != nil {
		return false, fmt.Errorf("unable to get messages: %w", err)
	}

	for _, msg := range msgs {
		if err = d.Session.DeleteMessage(d.channels[mainChannel], msg.ID, api.AuditLogReason("clearing")); err != nil {
			return false, fmt.Errorf("unable to delete message: %w", err)
		}
	}

	msgs, err = d.Session.Messages(d.channels[mainChannel], 1)
	if err != nil {
		return false, fmt.Errorf("unable to get messages: %w", err)
	}
	return len(msgs) == 0, nil
}

func (d *Discord) SendMessage(msgData api.SendMessageData, chid discord.ChannelID) error {
	_, err := d.Session.SendMessageComplex(d.channels[mainChannel], msgData)
	if err != nil {
		return err
	}
	d.Logger.Debug("sent message", zap.Uint64("channel id", uint64(chid)))
	return nil
}

func (d *Discord) NewMainMessage(msgData api.SendMessageData) error {
	var clearedAll bool
	var err error

	for !clearedAll {
		if clearedAll, err = d.deleteOldMessages(d.channels[mainChannel], 100); err != nil {
			return err
		}
	}

	return d.SendMessage(msgData, d.channels[mainChannel])
}

func (d *Discord) UpdateMainMessage(data api.EditMessageData) error {
	msgs, err := d.Session.Messages(d.channels[mainChannel], 1)
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

	msg, err := d.Session.EditMessageComplex(d.channels[mainChannel], msgs[0].ID, data)
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
