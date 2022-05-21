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
	uploaderInterval = 1 * time.Minute
	updaterInterval  = uploaderInterval
	timeoutInterval  = 2 * time.Second

	defaultDBName = "ichor"
)

type Server struct {
	Dexcom  *dexcom.Client
	Discord *discgo.Discord
	Ghastly *ghastly.Client
	Store   store.Store
	Logger  *zap.Logger
}

type Config struct {
	DexcomAccount  string `yaml:"dexcomAccount"`
	DexcomPassword string `yaml:"dexcomPassword"`
	DiscordToken   string `yaml:"discordToken"`
	DiscordGuild   string `yaml:"discordGuild"`
	MongoURI       string `yaml:"mongoURI"`
	TrevenantAddr  string `yaml:"trevenantAddress"`
	Logger         *zap.Logger
}

func New(config Config) (*Server, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutInterval)
	defer cancel()

	ms, err := store.New(ctx, config.MongoURI, defaultDBName, config.Logger)
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

	conn, err := grpc.Dial(config.TrevenantAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	gh := ghastly.New(conn, config.Logger)

	config.Logger.Debug("finished server setup", zap.Any("config", config))

	return &Server{
		Dexcom:  dexcom,
		Discord: discgo,
		Ghastly: gh,
		Store:   ms,
		Logger:  config.Logger,
	}, nil
}

// TODO: Functions below need to be updated/refactored.

func (s *Server) RunDiscord() {
	ticker := time.NewTicker(uploaderInterval)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		trs, err := s.Store.ReadGlucose(context.Background(), time.Now().UTC().Add(-12*time.Hour), time.Now().UTC())
		if err != nil {
			s.Logger.Debug("unable to read glucose from store", zap.Error(err))
		}

		if len(trs) == 0 {
			continue
		}

		fr, err := s.Ghastly.GenerateDailyPlot(context.Background(), trs)
		if err != nil {
			s.Logger.Debug("unable to generate daily plot", zap.Error(err))
		}

		if fr.GetId() == "-1" {
			s.Logger.Debug("unable to generate daily plot")
		}

		fileReader, err := s.Store.ReadFile(context.Background(), fr.GetId())
		if err != nil {
			s.Logger.Debug("unable to read file", zap.Error(err))
		}

		if err := s.Store.DeleteFile(context.Background(), fr.GetId()); err != nil {
			s.Logger.Debug("unable to delete file", zap.Error(err))
		}

		s.Discord.UpdateMain(&trs[0], fr.GetName(), fileReader)
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
