package discgo

import (
	"context"
	"fmt"
	"iv2/gourgeist/defs"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
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

func (d *Discord) SendMessage(data defs.MessageData, chName string) (uint64, error) {
	msgData := d.marshalSendData(data)
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
	md := d.unmarshalSendData(*discordMsg)
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

	md := d.marshalSendData(data)
	ed := api.EditMessageData{
		Content:     option.NewNullableString(md.Content),
		Embeds:      &md.Embeds,
		Attachments: &[]discord.Attachment{},
	}

	_, err = d.Session.EditMessageComplex(d.channels[d.mainCh], discord.MessageID(d.mid), ed)
	return err
}

// marshalSendData transforms data of type defs.MessageData to api.SendMessageData
// which arikawa expects.
func (d *Discord) marshalSendData(data defs.MessageData) api.SendMessageData {
	embeds := make([]discord.Embed, 0)
	for _, embed := range data.Embeds {
		fields := make([]discord.EmbedField, 0)

		for _, field := range embed.Fields {
			dField := discord.EmbedField{
				Name:   field.Name,
				Value:  field.Value,
				Inline: field.Inline,
			}
			fields = append(fields, dField)
		}

		dEmbed := discord.Embed{
			Title:       embed.Title,
			Description: embed.Description,
			Fields:      fields,
		}
		if embed.Image != nil {
			URL := embed.Image.Filename
			if !strings.Contains(embed.Image.Filename, "https://cdn.discordapp.com/attachments") {
				URL = "attachment://" + URL
			}
			dEmbed.Image = &discord.EmbedImage{URL: URL}
		}

		embeds = append(embeds, dEmbed)
	}

	files := make([]sendpart.File, 0)
	for _, file := range data.Files {
		files = append(files, sendpart.File{
			Name:   file.Name,
			Reader: file.Reader,
		})
	}

	md := api.SendMessageData{
		Content: data.Content,
		Embeds:  embeds,
		Files:   files,
	}

	if data.MentionEveryone {
		md.AllowedMentions = &api.AllowedMentions{
			Parse: []api.AllowedMentionType{api.AllowEveryoneMention},
		}
	}

	return md
}

// unmarshalSendData transforms data of type discord.Message to defs.MessageData.
func (d *Discord) unmarshalSendData(data discord.Message) defs.MessageData {
	embeds := make([]defs.EmbedData, 0)
	for _, embed := range data.Embeds {
		fields := make([]defs.EmbedField, 0)

		for _, field := range embed.Fields {
			dField := defs.EmbedField{
				Name:   field.Name,
				Value:  field.Value,
				Inline: field.Inline,
			}
			fields = append(fields, dField)
		}

		dEmbed := defs.EmbedData{
			Title:       embed.Title,
			Description: embed.Description,
			Fields:      fields,
		}
		if embed.Image != nil {
			dEmbed.Image = &defs.ImageData{
				Filename: strings.ReplaceAll(embed.Image.URL, "attachment://", ""),
			}
		}

		embeds = append(embeds, dEmbed)
	}

	md := defs.MessageData{
		Content: data.Content,
		Embeds:  embeds,
	}

	return md
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
