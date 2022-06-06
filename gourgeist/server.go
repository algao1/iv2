package gourgeist

import (
	"context"
	"iv2/gourgeist/dexcom"
	"iv2/gourgeist/discgo"
	"iv2/gourgeist/ghastly"
	"iv2/gourgeist/mongo"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	DownloaderInterval = 1 * time.Minute
	UpdaterInterval    = DownloaderInterval

	timeoutInterval = 2 * time.Second

	defaultDBName = "ichor"
)

type Config struct {
	DexcomAccount  string `yaml:"dexcomAccount"`
	DexcomPassword string `yaml:"dexcomPassword"`
	DiscordToken   string `yaml:"discordToken"`
	DiscordGuild   string `yaml:"discordGuild"`
	MongoURI       string `yaml:"mongoURI"`
	TrevenantAddr  string `yaml:"trevenantAddress"`
	Timezone       string `yaml:"timezone"`
	Logger         *zap.Logger
}

func Run(config Config) {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutInterval)
	defer cancel()

	var err error

	loc := time.Local
	if config.Timezone != "" {
		loc, err = time.LoadLocation(config.Timezone)
		if err != nil {
			panic(err)
		}
	}

	ms, err := mongo.New(ctx, config.MongoURI, defaultDBName, config.Logger)
	if err != nil {
		panic(err)
	}

	dexcom := dexcom.New(config.DexcomAccount, config.DexcomPassword, config.Logger)

	discgo, err := discgo.New(
		config.DiscordToken,
		config.Logger,
		loc,
	)
	if err != nil {
		panic(err)
	}

	ch := CommandHandler{Display: discgo, Store: ms, Logger: config.Logger, Location: loc}

	err = discgo.Setup(config.DiscordGuild, true, ch.InteractionCreateHandler())
	if err != nil {
		panic(err)
	}

	conn, err := grpc.Dial(config.TrevenantAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	gh := ghastly.New(conn, config.Logger)

	pu := PlotUpdater{
		Display:  discgo,
		Plotter:  gh,
		Store:    ms,
		Logger:   config.Logger,
		Location: loc,
	}

	f := Fetcher{
		Source: dexcom,
		Store:  ms,
		Logger: config.Logger,
	}

	go ExecuteTask(DownloaderInterval, func() error { return f.FetchAndLoad() }, config.Logger)
	ExecuteTask(DownloaderInterval, func() error { return pu.Update() }, config.Logger)
}

func ExecuteTask(interval time.Duration, task func() error, logger *zap.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for ; true; <-ticker.C {
		err := task()
		if err != nil {
			logger.Debug("error executing task", zap.Error(err))
		}
	}
}
