package gourgeist

import (
	"context"
	"fmt"
	"iv2/gourgeist/defs"
	"iv2/gourgeist/pkg/discgo"
	"iv2/gourgeist/pkg/mg"
	"time"

	"go.uber.org/zap"
)

type AnalyzerStore interface {
	mg.GlucoseStore
	mg.InsulinStore
	mg.AlertStore
}

type Analyzer struct {
	Messager discgo.Messager
	Store    AnalyzerStore

	Logger        *zap.Logger
	Location      *time.Location
	GlucoseConfig defs.GlucoseConfig
	AlarmConfig   defs.AlarmConfig
}

func (an *Analyzer) Run() error {
	checks := map[string]func() error{
		"glucose": an.AnalyzeGlucose,
		"insulin": an.AnalyzeInsulin,
	}
	for name, check := range checks {
		if err := check(); err != nil {
			an.Logger.Debug(
				"unable to complete check",
				zap.String("check", name),
				zap.Error(err),
			)
		}
	}
	return nil
}

func (an *Analyzer) AnalyzeGlucose() error {
	ctx := context.Background()
	now, start := time.Now(), time.Now().Add(defs.LookbackInterval)

	glucose, err := an.Store.ReadGlucose(ctx, start, now)
	if err != nil {
		return err
	} else if len(glucose) == 0 {
		return nil
	}

	// If we get an error, assume no previous alerts were sent.
	alertStart := now.Add(time.Duration(-1 * an.AlarmConfig.GlucoseTimeout * int(time.Minute)))
	alerts, _ := an.Store.ReadAlerts(ctx, alertStart, now)

	lowAlert, highAlert := true, true
	for _, alert := range alerts {
		switch alert.Label {
		case defs.LowGlucoseLabel:
			lowAlert = false
		case defs.HighGlucoseLabel:
			highAlert = false
		}
	}

	recentVal := glucose[len(glucose)-1].Mmol
	if recentVal >= an.GlucoseConfig.High && highAlert {
		return an.genAndSendAlert(
			defs.HighGlucoseLabel,
			fmt.Sprintf("current value: %.2f ≥ %.2f", recentVal, an.GlucoseConfig.High),
		)
	} else if recentVal <= an.GlucoseConfig.Low && lowAlert {
		return an.genAndSendAlert(
			defs.LowGlucoseLabel,
			fmt.Sprintf("current value: %.2f ≤ %.2f", recentVal, an.GlucoseConfig.Low),
		)
	}

	return nil
}

func (an *Analyzer) AnalyzeInsulin() error {
	// TODO: Need to make this check configurable.
	ctx := context.Background()
	now, start := time.Now(), time.Now().Add(-24*time.Hour)

	ins, err := an.Store.ReadInsulin(ctx, start, now)
	if err != nil {
		return err
	}

	missingAlert := true
	for _, in := range ins {
		if in.Type == defs.SlowActing.String() {
			missingAlert = false
		}
	}

	alertStart := now.Add(time.Duration(-1 * an.AlarmConfig.NoInsulinTimeout * int(time.Minute)))
	alerts, _ := an.Store.ReadAlerts(ctx, alertStart, now)
	for _, alert := range alerts {
		switch alert.Label {
		case defs.MissingSlowInsulinLabel:
			missingAlert = false
		}
	}

	if missingAlert {
		return an.genAndSendAlert(
			defs.MissingSlowInsulinLabel,
			fmt.Sprintf("last administered: ≥ %d hours ago", 24),
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

	_, err = an.Messager.SendMessage(defs.MessageData{
		Content:         fmt.Sprintln("⚠️ "+label) + fmt.Sprintln(reason) + "@everyone",
		MentionEveryone: true,
	}, defs.AlertsChannel)
	if err != nil {
		return err
	}

	return nil
}
