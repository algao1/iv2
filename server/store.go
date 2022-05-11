package server

import (
	"context"
	"iv2/dexcom"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const (
	dbName            = "ichor"
	glucoseCollection = "glucose"
)

type TimePoint interface {
	GetTime() time.Time
}

type Store struct {
	Client *mongo.Client
	Logger *zap.Logger
}

func (s *Store) writeEvent(ctx context.Context, collection string, event TimePoint) error {
	s.Logger.Debug("inserting event",
		zap.String("collection", collection),
		zap.Any("event", event))

	_, err := s.Client.
		Database(dbName).
		Collection(collection).
		UpdateOne(ctx, bson.M{
			"time": event.GetTime(),
		}, bson.M{"$set": event}, options.Update().SetUpsert(true))

	if err != nil {
		s.Logger.Debug("failed to insert event",
			zap.String("collection", collection),
			zap.Any("event", event),
			zap.Error(err))
	}

	return err
}

func (s *Store) WriteGlucose(ctx context.Context, tr *dexcom.TransformedReading) error {
	return s.writeEvent(ctx, glucoseCollection, tr)
}
