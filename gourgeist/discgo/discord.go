package discgo

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/session"
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
	UpdateMainMessage(msgData api.SendMessageData) error
}

func New(token string, logger *zap.Logger, loc *time.Location) (*Discord, error) {
	ses, err := session.NewWithIntents("Bot "+token, gateway.IntentGuildMessages)
	if err != nil {
		return nil, err
	}

	err = ses.Open()
	if err != nil {
		return nil, err
	}

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

	channels, err := d.Session.Channels(d.gid)
	if err != nil {
		return err
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
		return err
	}

	d.chid = channel.ID
	return nil
}

func (d *Discord) GetMainMessage() (*discord.Message, error) {
	msgs, err := d.Session.Messages(d.chid, 10)
	if err != nil {
		return nil, err
	}
	if len(msgs) == 0 {
		return nil, fmt.Errorf("no messages found")
	}
	return &msgs[0], nil
}

func (d *Discord) UpdateMainMessage(msgData api.SendMessageData) error {
	msgs, err := d.Session.Messages(d.chid, 10)
	if err != nil {
		return err
	}

	for _, msg := range msgs {
		err = d.Session.DeleteMessage(d.chid, msg.ID)
		if err != nil {
			return err
		}
	}

	d.Logger.Debug("updating main message", zap.Any("msgData", msgData))

	_, err = d.Session.SendMessageComplex(d.chid, msgData)
	return err
}
