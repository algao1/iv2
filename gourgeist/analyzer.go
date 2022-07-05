package gourgeist

import (
	"context"
	"fmt"
	"iv2/gourgeist/defs"
	"iv2/gourgeist/discgo"
	"iv2/gourgeist/mg"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"go.uber.org/zap"
)

const (
	HighGlucoseLabel = "High Glucose"
	LowGlucoseLabel  = "Low Glucose"
)

type AnalyzerStore interface {
	mg.GlucoseStore
	mg.AlertStore
}

type Analyzer struct {
	Messager discgo.Messager
	Store    AnalyzerStore

	Logger        *zap.Logger
	Location      *time.Location
	GlucoseConfig defs.GlucoseConfig
}

func (an *Analyzer) AnalyzeGlucose() error {
	ctx := context.Background()
	now := time.Now()
	start := now.Add(lookbackInterval)

	glucose, err := an.Store.ReadGlucose(ctx, start, now)
	if err != nil {
		return err
	}

	if len(glucose) == 0 {
		return nil
	}

	// TODO: Can probably add a filter to search by label.
	alertStart := now.Add(-1 * time.Hour)
	alerts, _ := an.Store.ReadAlerts(ctx, alertStart, now)
	lowAlert, highAlert := true, true
	for _, alert := range alerts {
		switch alert.Label {
		case LowGlucoseLabel:
			lowAlert = false
		case HighGlucoseLabel:
			highAlert = false
		}
	}

	recentVal := glucose[len(glucose)-1].Mmol

	if recentVal >= an.GlucoseConfig.High && highAlert {
		return an.genAndSendAlert(
			HighGlucoseLabel,
			fmt.Sprintf("current value: %.2f ≥ %.2f", recentVal, an.GlucoseConfig.High),
		)
	} else if recentVal <= an.GlucoseConfig.Low && lowAlert {
		return an.genAndSendAlert(
			LowGlucoseLabel,
			fmt.Sprintf("current value: %.2f ≤ %.2f", recentVal, an.GlucoseConfig.Low),
		)
	}

	return nil
}

func (an *Analyzer) genAndSendAlert(label, reason string) error {
	_, err := an.Store.WriteAlert(context.Background(), &defs.Alert{
		Time:   time.Now(),
		Label:  label,
		Reason: reason,
	})
	if err != nil {
		return err
	}

	embed := discord.Embed{
		Fields: []discord.EmbedField{
			{
				Name:  "⚠️ " + label,
				Value: reason,
			},
		},
	}

	_, err = an.Messager.SendMessage(api.SendMessageData{
		Content: "@everyone",
		Embeds:  []discord.Embed{embed},
		AllowedMentions: &api.AllowedMentions{
			Parse: []api.AllowedMentionType{api.AllowEveryoneMention},
		},
	}, alertsChannel)
	if err != nil {
		return err
	}

	return nil
}
