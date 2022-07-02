package gourgeist

import (
	"context"
	"iv2/gourgeist/defs"
	"iv2/gourgeist/dexcom"
	"iv2/gourgeist/discgo"
	"iv2/gourgeist/ghastly"
	"iv2/gourgeist/mg"
	"strconv"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	DownloaderInterval = 1 * time.Minute
	UpdaterInterval    = DownloaderInterval
	timeoutInterval    = 2 * time.Second

	alertsChannel = "alerts"

	defaultDBName = "ichor"
)

func Run(cfg defs.Config) {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutInterval)
	defer cancel()

	var err error

	loc := time.Local
	if cfg.Timezone != "" {
		loc, err = time.LoadLocation(cfg.Timezone)
		if err != nil {
			panic(err)
		}
	}

	ms, err := mg.New(ctx, cfg.Mongo, defaultDBName, cfg.Logger)
	if err != nil {
		panic(err)
	}

	dexcom := dexcom.New(cfg.Dexcom.Account, cfg.Dexcom.Password, cfg.Logger)

	dg, err := discgo.New(
		cfg.Discord.Token,
		cfg.Logger,
		loc,
	)
	if err != nil {
		panic(err)
	}

	ch := CommandHandler{Display: dg, Store: ms, Logger: cfg.Logger, Location: loc}

	err = dg.Setup(
		strconv.Itoa(cfg.Discord.Guild),
		discgo.Commands,
		[]string{alertsChannel},
		ch.InteractionCreateHandler(),
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

	pu := PlotUpdater{
		Display:       dg,
		Plotter:       gh,
		Store:         ms,
		Logger:        cfg.Logger,
		Location:      loc,
		GlucoseConfig: cfg.Glucose,
	}

	an := Analyzer{
		Display:       dg,
		Store:         ms,
		Logger:        cfg.Logger,
		Location:      loc,
		GlucoseConfig: cfg.Glucose,
	}

	f := Fetcher{
		Source: dexcom,
		Store:  ms,
		Logger: cfg.Logger,
	}

	go ExecuteTask("glucose-fetcher", DownloaderInterval, func() error { return f.FetchAndLoad() }, cfg.Logger)
	go ExecuteTask("glucose-plotter", DownloaderInterval, func() error { return pu.Update() }, cfg.Logger)
	ExecuteTask("glucose-analyzer", DownloaderInterval, func() error { return an.AnalyzeGlucose() }, cfg.Logger)
}

func ExecuteTask(taskName string, interval time.Duration, task func() error, logger *zap.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for ; true; <-ticker.C {
		err := task()
		if err != nil {
			logger.Error("error executing task", zap.String("task", taskName), zap.Error(err))
		}
	}
}
