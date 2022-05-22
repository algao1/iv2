package gourgeist

import (
	"context"
	"iv2/gourgeist/dexcom"
	"iv2/gourgeist/discgo"
	"iv2/gourgeist/ghastly"
	"iv2/gourgeist/store"
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

type Server struct {
	Dexcom   *dexcom.Client
	Discord  *discgo.Discord
	Ghastly  *ghastly.Client
	Store    store.Store
	Logger   *zap.Logger
	Location *time.Location
}

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

func New(config Config) (*Server, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutInterval)
	defer cancel()

	var err error

	loc := time.Local
	if config.Timezone != "" {
		loc, err = time.LoadLocation(config.Timezone)
		if err != nil {
			return nil, err
		}
	}

	ms, err := store.New(ctx, config.MongoURI, defaultDBName, config.Logger)
	if err != nil {
		return nil, err
	}

	dexcom := dexcom.New(config.DexcomAccount, config.DexcomPassword, config.Logger)

	discgo, err := discgo.New(config.DiscordToken, config.Logger, loc)
	if err != nil {
		return nil, err
	}
	err = discgo.Setup(config.DiscordGuild)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.Dial(config.TrevenantAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	gh := ghastly.New(conn, config.Logger)

	config.Logger.Debug("finished server setup", zap.Any("config", config))

	return &Server{
		Dexcom:   dexcom,
		Discord:  discgo,
		Ghastly:  gh,
		Store:    ms,
		Logger:   config.Logger,
		Location: loc,
	}, nil
}

// TODO: Below needs to be refactored into an executor struct.

func (s *Server) ExecuteTask(interval time.Duration, task func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for ; true; <-ticker.C {
		task()
	}
}

func (s *Server) UpdateDiscord() {
	du := DisplayUpdater{Display: s.Discord, Plotter: s.Ghastly, Store: s.Store, Logger: s.Logger, Location: s.Location}
	du.Update()
}

func (s *Server) FetchUploadReadings() {
	f := Fetcher{Source: s.Dexcom, Store: s.Store, Logger: s.Logger}
	f.FetchAndLoad()
}
