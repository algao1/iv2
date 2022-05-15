package discgo

import (
	"iv2/server/dexcom"
	"strconv"
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
	Session *session.Session
	Logger  *zap.Logger

	gid  discord.GuildID
	chid discord.ChannelID
}

func New(token string, logger *zap.Logger) (*Discord, error) {
	ses, err := session.NewWithIntents("Bot "+token, gateway.IntentGuildMessages)
	if err != nil {
		return nil, err
	}

	err = ses.Open()
	if err != nil {
		return nil, err
	}

	return &Discord{
		Session: ses,
		Logger:  logger,
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

func floatToString(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

func (d *Discord) UpdateMain(tr *dexcom.TransformedReading) error {
	msgs, err := d.Session.Messages(d.chid, 10)
	if err != nil {
		return err
	}

	embed := discord.Embed{
		Title: time.Now().Format(TimeFormat),
		Fields: []discord.EmbedField{
			{Name: "Current", Value: floatToString(tr.Mmol)},
		},
	}

	if len(msgs) == 0 {
		_, err = d.Session.SendEmbed(d.chid, embed)
		return err
	}
	_, err = d.Session.EditMessage(d.chid, msgs[0].ID, "", &embed, false)

	d.Logger.Debug("updated main message", zap.Any("embed", embed))

	return err
}
