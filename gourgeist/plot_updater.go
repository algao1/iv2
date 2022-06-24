package gourgeist

import (
	"context"
	"fmt"
	"iv2/gourgeist/discgo"
	"iv2/gourgeist/ghastly"
	"iv2/gourgeist/mg"
	"strconv"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
	"go.uber.org/zap"
)

const lookbackInterval = -12 * time.Hour

type PlotUpdater struct {
	Display discgo.Display
	Plotter ghastly.Plotter
	Store   mg.Store

	Logger   *zap.Logger
	Location *time.Location
}

func (pu PlotUpdater) Update() error {
	pd, err := pu.getPlotData()
	if err != nil {
		return fmt.Errorf("unable to get plot data: %w", err)
	}

	if len(pd.Glucose) == 0 {
		return fmt.Errorf("no glucose readings found")
	}

	prevMsg, err := pu.Display.GetMainMessage()
	if err != nil {
		pu.Logger.Debug("unable to get main message", zap.Error(err))
	}

	if prevMsg != nil && len(prevMsg.Embeds) > 0 &&
		prevMsg.Embeds[0].Title == pd.Glucose[len(pd.Glucose)-1].GetTime().In(pu.Location).Format(discgo.TimeFormat) {
		pu.Logger.Debug("skipping display update, up to date", zap.String("date", prevMsg.Embeds[0].Title))
		return nil
	}

	fr, err := pu.Plotter.GenerateDailyPlot(context.Background(), pd)
	if err != nil {
		pu.Logger.Debug("unable to generate daily plot", zap.Error(err))
	}

	fileReader, err := pu.Store.ReadFile(context.Background(), fr.GetId())
	if err != nil {
		pu.Logger.Debug("unable to read file", zap.Error(err))
	}

	if err := pu.Store.DeleteFile(context.Background(), fr.GetId()); err != nil {
		pu.Logger.Debug("unable to delete file", zap.Error(err))
	}

	glucose := pd.Glucose[len(pd.Glucose)-1]
	embed := discord.Embed{
		Title: glucose.Time.In(pu.Location).Format(discgo.TimeFormat),
		Fields: []discord.EmbedField{
			{Name: "Current", Value: strconv.FormatFloat(glucose.Mmol, 'f', 2, 64), Inline: true},
			{Name: "Trend", Value: glucose.Trend, Inline: true},
		},
	}

	desc, err := newDescription(pu.Store, pu.Location)
	if err == nil {
		embed.Description = desc
	}

	msgData := api.SendMessageData{
		Embeds: []discord.Embed{embed},
		Files:  []sendpart.File{},
	}

	if fileReader != nil {
		pu.Logger.Debug("adding image to embed", zap.String("name", fr.GetName()))
		msgData.Embeds[0].Image = &discord.EmbedImage{URL: "attachment://" + fr.GetName()}
		msgData.Files = append(msgData.Files, sendpart.File{Name: fr.GetName(), Reader: fileReader})
	}

	return pu.Display.NewMainMessage(msgData)
}

func (pu *PlotUpdater) getPlotData() (*ghastly.PlotData, error) {
	end := time.Now()
	start := end.Add(lookbackInterval)
	ctx := context.Background()

	glucose, err := pu.Store.ReadGlucose(ctx, start, end)
	if err != nil {
		return nil, err
	}

	carbs, err := pu.Store.ReadCarbs(ctx, start, end)
	if err != nil {
		return nil, err
	}

	insulin, err := pu.Store.ReadInsulin(ctx, start, end)
	if err != nil {
		return nil, err
	}

	return &ghastly.PlotData{Glucose: glucose, Carbs: carbs, Insulin: insulin}, nil
}
