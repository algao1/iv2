package discgo

import (
	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/session"
	"go.uber.org/zap"
)

const (
	broadcastChannelName = "iv2"
)

type Discord struct {
	Session *session.Session
	Logger  *zap.Logger

	gid discord.GuildID
}

func New(token, guildID string, logger *zap.Logger) (*Discord, error) {
	ses, err := session.NewWithIntents("Bot "+token, gateway.IntentGuildMessages)
	if err != nil {
		return nil, err
	}

	err = ses.Open()
	if err != nil {
		return nil, err
	}

	sf, err := discord.ParseSnowflake(guildID)
	if err != nil {
		return nil, err
	}
	gid := discord.GuildID(sf)

	return &Discord{
		Session: ses,
		Logger:  logger,
		gid:     gid,
	}, nil
}

func (d *Discord) Setup() error {
	channels, err := d.Session.Channels(d.gid)
	if err != nil {
		return err
	}

	for _, ch := range channels {
		if ch.Type == discord.GuildText && ch.Name == broadcastChannelName {
			return nil
		}
	}

	d.Logger.Debug("creating channel", zap.String("channel name", broadcastChannelName))

	_, err = d.Session.CreateChannel(d.gid, api.CreateChannelData{
		Name: broadcastChannelName,
		Type: discord.GuildText,
	})
	return err
}
