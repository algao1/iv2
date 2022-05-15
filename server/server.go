package server

import (
	"context"
	"iv2/server/dexcom"
	"iv2/server/discgo"
	"iv2/server/store"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const (
	uploaderInterval = 1 * time.Minute
	updaterInterval  = uploaderInterval
	timeoutInterval  = 2 * time.Second
)

type Server struct {
	Dexcom  *dexcom.Client
	Discord *discgo.Discord
	Store   store.Store
	Logger  *zap.Logger
}

type Config struct {
	DexcomAccount  string `yaml:"dexcomAccount"`
	DexcomPassword string `yaml:"dexcomPassword"`
	DiscordToken   string `yaml:"discordToken"`
	DiscordGuild   string `yaml:"discordGuild"`
	MongoURI       string `yaml:"mongoURI"`
	Logger         *zap.Logger
}

func New(config Config) (*Server, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutInterval)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(config.MongoURI))
	if err != nil {
		return nil, err
	}

	dexcom := dexcom.New(config.DexcomAccount, config.DexcomPassword, config.Logger)

	discgo, err := discgo.New(config.DiscordToken, config.Logger)
	if err != nil {
		return nil, err
	}
	err = discgo.Setup(config.DiscordGuild)
	if err != nil {
		return nil, err
	}

	ms := &store.MongoStore{Client: mongoClient, Logger: config.Logger}

	config.Logger.Debug("finished server setup", zap.Any("config", config))

	return &Server{
		Dexcom:  dexcom,
		Discord: discgo,
		Store:   ms,
		Logger:  config.Logger,
	}, nil
}

func (s *Server) RunDiscord() {
	ticker := time.NewTicker(uploaderInterval)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		trs, err := s.Store.ReadGlucose(context.Background(), time.Now().Add(-10*time.Minute), time.Now())
		if err != nil {
			s.Logger.Debug("unable to read glucose from store", zap.Error(err))
		}

		if len(trs) == 0 {
			continue
		}

		s.Discord.UpdateMain(&trs[len(trs)-1])
	}
}

func (s *Server) RunUploader() {
	ticker := time.NewTicker(uploaderInterval)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		trs, _ := s.Dexcom.Readings(context.Background(), dexcom.MinuteLimit, dexcom.CountLimit)
		for _, tr := range trs {
			exist, err := s.Store.WriteGlucose(context.Background(), tr)
			if err != nil {
				s.Logger.Debug("unable to write glucose to store", zap.Error(err))
			}

			if exist {
				break
			}
		}
	}
}
