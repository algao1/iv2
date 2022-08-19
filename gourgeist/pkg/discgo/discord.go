package discgo

import (
	"context"
	"fmt"
	"iv2/gourgeist/defs"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
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
	mid      uint64 // Main message ID.
	mainCh   string
	channels map[string]discord.ChannelID
}

type Display interface {
	Messager
	Interactioner
}

type Messager interface {
	SendMessage(data defs.MessageData, chName string) (uint64, error)
	GetMainMessage() (*defs.MessageData, error)
	NewMainMessage(data defs.MessageData) error
	UpdateMainMessage(data defs.MessageData) error
}

type Interactioner interface {
	RespondInteraction(id uint64, token string, resp defs.InteractionResponse) error
	DeleteInteractionResponse(appID uint64, token string) error
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

func (d *Discord) SendMessage(data defs.MessageData, chName string) (uint64, error) {
	msgData := marshalSendData(data)
	msg, err := d.Session.SendMessageComplex(d.channels[chName], msgData)
	if err != nil {
		return 0, err
	}
	d.Logger.Debug("sent message", zap.String("channel name", chName))
	return uint64(msg.ID), nil
}

func (d *Discord) GetMainMessage() (*defs.MessageData, error) {
	discordMsg, err := d.Session.Message(d.channels[d.mainCh], discord.MessageID(d.mid))
	if err != nil {
		return nil, err
	}
	md := unmarshalMessage(*discordMsg)
	return &md, nil
}

func (d *Discord) NewMainMessage(data defs.MessageData) error {
	err := d.deleteMessages(d.channels[d.mainCh], 0)
	if err != nil {
		return err
	}

	messageID, err := d.SendMessage(data, d.mainCh)
	if err != nil {
		return err
	}
	d.mid = messageID

	return nil
}

func (d *Discord) UpdateMainMessage(data defs.MessageData) error {
	err := d.deleteMessages(d.channels[d.mainCh], discord.MessageID(d.mid))
	if err != nil {
		return err
	}

	msg, err := d.GetMainMessage()
	if err != nil {
		return err
	} else if msg == nil {
		return d.NewMainMessage(data)
	}

	md := marshalSendData(data)
	ed := api.EditMessageData{
		Content:     option.NewNullableString(md.Content),
		Embeds:      &md.Embeds,
		Attachments: &[]discord.Attachment{},
	}

	_, err = d.Session.EditMessageComplex(d.channels[d.mainCh], discord.MessageID(d.mid), ed)
	return err
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

func (d *Discord) RespondInteraction(id uint64, token string, resp defs.InteractionResponse) error {
	// Currently only supports content replies.
	sendData := marshalSendData(resp.Data)
	msgResp := api.InteractionResponse{
		Type: api.MessageInteractionWithSource,
		Data: &api.InteractionResponseData{
			Content: option.NewNullableString(sendData.Content),
		},
	}
	return d.Session.RespondInteraction(discord.InteractionID(id), token, msgResp)
}

func (d *Discord) DeleteInteractionResponse(appID uint64, token string) error {
	return d.Session.DeleteInteractionResponse(discord.AppID(appID), token)
}
