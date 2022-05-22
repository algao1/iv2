package gourgeist

import (
	"context"
	"fmt"
	"iv2/gourgeist/discgo"
	"iv2/gourgeist/ghastly"
	"iv2/gourgeist/store"
	"strconv"
	"time"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"go.uber.org/zap"
)

const lookbackInterval = -12 * time.Hour

type DisplayUpdater struct {
	Display discgo.Display
	Plotter ghastly.Plotter
	Store   store.Store

	Logger   *zap.Logger
	Location *time.Location
}

func (u DisplayUpdater) Update() error {
	now := time.Now()
	trs, err := u.Store.ReadGlucose(context.Background(), now.Add(lookbackInterval), time.Now())
	if err != nil {
		return fmt.Errorf("unable to read glucose from store: %w", err)
	}

	if len(trs) == 0 {
		return fmt.Errorf("no glucose readings found")
	}

	fr, err := u.Plotter.GenerateDailyPlot(context.Background(), trs)
	if err != nil {
		u.Logger.Debug("unable to generate daily plot", zap.Error(err))
	}

	fileReader, err := u.Store.ReadFile(context.Background(), fr.GetId())
	if err != nil {
		u.Logger.Debug("unable to read file", zap.Error(err))
	}

	if err := u.Store.DeleteFile(context.Background(), fr.GetId()); err != nil {
		u.Logger.Debug("unable to delete file", zap.Error(err))
	}

	tr := trs[0]
	embed := discord.Embed{
		Title: tr.Time.In(u.Location).Format(discgo.TimeFormat),
		Fields: []discord.EmbedField{
			{Name: "Current", Value: strconv.FormatFloat(tr.Mmol, 'f', 2, 64)},
		},
	}
	msgData := api.SendMessageData{
		Embed: &embed,
		Files: []api.SendMessageFile{},
	}

	if fileReader != nil {
		u.Logger.Debug("adding image to embed", zap.String("name", fr.GetName()))
		embed.Image = &discord.EmbedImage{URL: "attachment://" + fr.GetName()}
		msgData.Files = append(msgData.Files, api.SendMessageFile{Name: fr.GetName(), Reader: fileReader})
	}

	return u.Display.UpdateMainMessage(msgData)
}
