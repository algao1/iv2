package gourgeist

import (
	"context"
	"fmt"
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

type Gourgeist struct {
	commandHandler commander.CommandHandler
	fetcher        Fetcher
	plotUpdater    PlotUpdater
	analyzer       Analyzer
	logger         *zap.Logger
}

func NewGourgeist(cfg defs.Config) (*Gourgeist, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defs.TimeoutInterval)
	defer cancel()

	fmt.Println(cfg)

	loc := time.Local
	if cfg.Timezone != "" {
		lloc, err := time.LoadLocation(cfg.Timezone)
		if err != nil {
			return nil, fmt.Errorf("unable to parse location: %w", err)
		}
		loc = lloc
	}

	ms, err := mg.New(ctx, cfg.Mongo, defs.DefaultDB, cfg.Logger)
	if err != nil {
		return nil, fmt.Errorf("unable to create store: %w", err)
	}

	dexcom := dexcom.New(cfg.Dexcom.Account, cfg.Dexcom.Password, cfg.Logger)
	f := Fetcher{Source: dexcom, Store: ms, Logger: cfg.Logger}

	guildID := strconv.Itoa(cfg.Discord.Guild)
	dg, err := discgo.New(cfg.Discord.Token, guildID, cfg.Logger, loc)
	if err != nil {
		return nil, fmt.Errorf("unable to create discord link: %w", err)
	}

	conn, err := grpc.Dial(cfg.TrevenantAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("unable to dial: %w", err)
	}
	gh := ghastly.New(conn, cfg.Logger)

	ch := commander.CommandHandler{
		Display:       dg,
		Plotter:       gh,
		Store:         ms,
		Logger:        cfg.Logger,
		Descriptor:    dcr.New(loc),
		Location:      loc,
		GlucoseConfig: cfg.Glucose,
	}

	if err = dg.Setup([]string{defs.AlertsChannel, defs.ReportsChannel}, ch.CreateHandler()); err != nil {
		return nil, fmt.Errorf("unable to setup discord link: %w", err)
	}

	pu := PlotUpdater{
		Messager:      dg,
		Plotter:       gh,
		Store:         ms,
		Logger:        cfg.Logger,
		Descriptor:    dcr.New(loc),
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

	g := &Gourgeist{
		commandHandler: ch,
		fetcher:        f,
		plotUpdater:    pu,
		analyzer:       an,
		logger:         cfg.Logger,
	}
	g.run()

	return g, nil
}

func (g *Gourgeist) run() {
	ticker := time.NewTicker(defs.DownloaderInterval)
	defer ticker.Stop()
	for ; true; <-ticker.C {
		if err := g.fetcher.FetchAndLoad(); err != nil {
			g.logger.Error("fetching error", zap.Error(err))
		}
		if err := g.plotUpdater.Update(); err != nil {
			g.logger.Error("plot update error", zap.Error(err))
		}
		if err := g.analyzer.Run(); err != nil {
			g.logger.Error("analyzer error", zap.Error(err))
		}
	}
}
