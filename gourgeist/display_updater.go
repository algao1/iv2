package gourgeist

import (
	"context"
	"fmt"
	"iv2/gourgeist/discgo"
	"iv2/gourgeist/ghastly"
	"iv2/gourgeist/mongo"
	"strconv"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
	"go.uber.org/zap"
)

const lookbackInterval = -12 * time.Hour

type DisplayUpdater struct {
	Display discgo.Display
	Plotter ghastly.Plotter
	Store   mongo.Store

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

	prevMsg, err := u.Display.GetMainMessage()
	if err != nil {
		return err
	}

	if prevMsg != nil && len(prevMsg.Embeds) > 0 &&
		prevMsg.Embeds[0].Title == trs[0].GetTime().In(u.Location).Format(discgo.TimeFormat) {
		u.Logger.Debug("skipping display update, up to date", zap.String("date", prevMsg.Embeds[0].Title))
		return nil
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
		Embeds: []discord.Embed{embed},
		Files:  []sendpart.File{},
	}

	if fileReader != nil {
		u.Logger.Debug("adding image to embed", zap.String("name", fr.GetName()))
		msgData.Embeds[0].Image = &discord.EmbedImage{URL: "attachment://" + fr.GetName()}
		msgData.Files = append(msgData.Files, sendpart.File{Name: fr.GetName(), Reader: fileReader})
	}

	return u.Display.NewMainMessage(msgData)
}
