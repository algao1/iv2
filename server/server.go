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
	timeoutInterval  = 5 * time.Second
)

type Server struct {
	Dexcom *dexcom.Client
	Store  *store.Store
}

type ServerConfig struct {
	DexcomAccount  string
	DexcomPassword string
	MongoURI       string
	Logger         *zap.Logger
}

func New(config ServerConfig) (*Server, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutInterval)
	defer cancel()

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
