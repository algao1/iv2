package server

import (
	"context"
	"iv2/server/dexcom"
	"iv2/server/store"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const (
	uploaderInterval = 1 * time.Minute
	timeoutInterval  = 2 * time.Second
)

type Server struct {
	Dexcom *dexcom.Client
	Store  *store.Store
}

type Config struct {
	DexcomAccount  string `yaml:"dexcomAccount"`
	DexcomPassword string `yaml:"dexcomPassword"`
	MongoURI       string `yaml:"mongoURI"`
	Logger         *zap.Logger
}

func New(config Config) (*Server, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutInterval)
	defer cancel()

	config.Logger.Debug("connecting to mongo", zap.String("uri", config.MongoURI))

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(config.MongoURI))
	if err != nil {
		return nil, err
	}

	dexcom := dexcom.New(config.DexcomAccount, config.DexcomPassword, config.Logger)
	store := &store.Store{Client: mongoClient, Logger: config.Logger}

	return &Server{
		Dexcom: dexcom,
		Store:  store,
	}, nil
}

func (s *Server) RunUploader() {
	ticker := time.NewTicker(uploaderInterval)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		trs, _ := s.Dexcom.Readings(context.Background(), dexcom.MinuteLimit, dexcom.CountLimit)
		for _, tr := range trs {
			s.Store.WriteGlucose(context.Background(), tr)
		}
	}
}
