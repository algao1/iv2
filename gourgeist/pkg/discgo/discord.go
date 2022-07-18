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
	TimeFormat  = "2006-01-02 03:04 PM"
	mainChannel = "iv2"
	batchLimit  = 100
)

type Discord struct {
	Session  *session.Session
	Logger   *zap.Logger
	Location *time.Location

	// A little hackish to store all the data in a temporary cache.
	gid      discord.GuildID
	mid      discord.MessageID // Main message ID.
	mainCh   string
	channels map[string]discord.ChannelID
}

type Display interface {
	Messager
	Interactioner
}

type Messager interface {
	SendMessage(msgData api.SendMessageData, chName string) (discord.MessageID, error)
	GetMainMessage() (*discord.Message, error)
	NewMainMessage(msgData api.SendMessageData) error
	UpdateMainMessage(data api.EditMessageData) error
}

type Interactioner interface {
	RespondInteraction(id discord.InteractionID, token string, resp api.InteractionResponse) error
	DeleteInteractionResponse(appID discord.AppID, token string) error
}

func New(token, guildID string, logger *zap.Logger, loc *time.Location) (*Discord, error) {
	ses := session.NewWithIntents("Bot "+token, gateway.IntentGuildMessages)
	ses.AddIntents(gateway.IntentGuilds)
	ses.AddIntents(gateway.IntentGuildMessages)

	if err := ses.Open(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to open session: %w", err)
	}

	sf, err := discord.ParseSnowflake(guildID)
	if err != nil {
		return nil, err
	}

	return &Discord{
		Session:  ses,
		Logger:   logger,
		Location: loc,
		gid:      discord.GuildID(sf),
		mainCh:   mainChannel, // TODO: Maybe pass responsibility to user?
	}, nil
}

// TODO: Function signature is overloaded, need addressing.

func (d *Discord) Setup(cmds []api.CreateCommandData, channels []string, handlers ...interface{}) error {
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
	channels = append(channels, d.mainCh)
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

func (d *Discord) SendMessage(msgData api.SendMessageData, chName string) (discord.MessageID, error) {
	msg, err := d.Session.SendMessageComplex(d.channels[chName], msgData)
	if err != nil {
		return 0, err
	}
	d.Logger.Debug("sent message", zap.String("channel name", chName))
	return msg.ID, nil
}

func (d *Discord) GetMainMessage() (*discord.Message, error) {
	return d.Session.Message(d.channels[d.mainCh], d.mid)
}

func (d *Discord) NewMainMessage(msgData api.SendMessageData) error {
	err := d.deleteMessages(d.channels[d.mainCh], 0)
	if err != nil {
		return err
	}

	messageID, err := d.SendMessage(msgData, d.mainCh)
	if err != nil {
		return err
	}
	d.mid = messageID

	return nil
}

func (d *Discord) UpdateMainMessage(data api.EditMessageData) error {
	err := d.deleteMessages(d.channels[d.mainCh], d.mid)
	if err != nil {
		return err
	}

	msg, err := d.GetMainMessage()
	if err != nil {
		return err
	} else if msg == nil {
		msgData := api.SendMessageData{Content: data.Content.Val}
		if data.Embeds != nil {
			msgData.Embeds = *data.Embeds
		}
		return d.NewMainMessage(msgData)
	}

	_, err = d.Session.EditMessageComplex(d.channels[d.mainCh], d.mid, data)
	if err != nil {
		return fmt.Errorf("unable to edit main message: %w", err)
	}
	d.Logger.Debug("updated main message")

	return nil
}

func (d *Discord) deleteMessages(chid discord.ChannelID, exclude discord.MessageID) error {
	var clearedAll bool
	for !clearedAll {
		msgs, err := d.Session.Messages(d.channels[d.mainCh], batchLimit)
		if err != nil {
			return fmt.Errorf("unable to get messages: %w", err)
		}

		for _, msg := range msgs {
			if msg.ID == exclude {
				continue
			}
			if err = d.Session.DeleteMessage(d.channels[d.mainCh], msg.ID, api.AuditLogReason("clearing")); err != nil {
				return fmt.Errorf("unable to delete message: %w", err)
			}
		}

		if len(msgs) == 0 || len(msgs) == 1 && msgs[0].ID == exclude {
			break
		}
	}

	return nil
}

func (d *Discord) RespondInteraction(id discord.InteractionID, token string, resp api.InteractionResponse) error {
	return d.Session.RespondInteraction(id, token, resp)
}

func (d *Discord) DeleteInteractionResponse(appID discord.AppID, token string) error {
	return d.Session.DeleteInteractionResponse(appID, token)
}