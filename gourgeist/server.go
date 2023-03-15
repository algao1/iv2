package gourgeist

import (
	"context"
	"iv2/gourgeist/commander"
	"iv2/gourgeist/defs"
	dcr "iv2/gourgeist/pkg/desc"
	"iv2/gourgeist/pkg/dexcom"
	"iv2/gourgeist/pkg/discgo"
	"iv2/gourgeist/pkg/ghastly"
	"iv2/gourgeist/pkg/mg"
	"strconv"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Run(cfg defs.Config) {
	ctx, cancel := context.WithTimeout(context.Background(), defs.TimeoutInterval)
	defer cancel()

	var err error

	loc := time.Local
	if cfg.Timezone != "" {
		loc, err = time.LoadLocation(cfg.Timezone)
		if err != nil {
			panic(err)
		}
	}

	ms, err := mg.New(ctx, cfg.Mongo, defs.DefaultDB, cfg.Logger)
	if err != nil {
		panic(err)
	}

	dexcom := dexcom.New(
		cfg.Dexcom.Account,
		cfg.Dexcom.Password,
		cfg.Logger,
	)

	dg, err := discgo.New(
		cfg.Discord.Token,
		strconv.Itoa(cfg.Discord.Guild),
		cfg.Logger,
		loc,
	)
	if err != nil {
		panic(err)
	}

	conn, err := grpc.Dial(
		cfg.TrevenantAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}
	gh := ghastly.New(conn, cfg.Logger)

	d := dcr.New(loc)

	ch := commander.CommandHandler{
		Display:       dg,
		Plotter:       gh,
		Store:         ms,
		Logger:        cfg.Logger,
		Descriptor:    d,
		Location:      loc,
		GlucoseConfig: cfg.Glucose,
	}

	if err = dg.Setup(
		[]string{defs.AlertsChannel, defs.ReportsChannel},
		ch.CreateHandler(),
	); err != nil {
		panic(err)
	}

	pu := PlotUpdater{
		Messager:      dg,
		Plotter:       gh,
		Store:         ms,
		Logger:        cfg.Logger,
		Descriptor:    d,
		Location:      loc,
		GlucoseConfig: cfg.Glucose,
	}

	an := Analyzer{
		Messager:      dg,
		Store:         ms,
		Logger:        cfg.Logger,
		Location:      loc,
		GlucoseConfig: cfg.Glucose,
		AlarmConfig:   cfg.Alarm,
	}

	f := Fetcher{
		Source: dexcom,
		Store:  ms,
		Logger: cfg.Logger,
	}

	// TODO: Eventually, separate this out to be triggered by updates
	// so that they don't run constantly.
	ExecuteTask("loop", defs.DownloaderInterval, func() error {
		var err error
		if err = f.FetchAndLoad(); err != nil {
			cfg.Logger.Error("fetching error", zap.Error(err))
		}
		if err = pu.Update(); err != nil {
			cfg.Logger.Error("plot update error", zap.Error(err))
		}
		if err = an.Run(); err != nil {
			cfg.Logger.Error("analyzer error", zap.Error(err))
		}
		return nil
	}, cfg.Logger)
}

func ExecuteTask(taskName string, interval time.Duration, task func() error, logger *zap.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for ; true; <-ticker.C {
		err := task()
		if err != nil {
			logger.Error(
				"error executing task",
				zap.String("task", taskName),
				zap.Error(err),
			)
		}
	}
}
