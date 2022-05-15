package store

import (
	"context"
	"iv2/server/dexcom"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

type Store interface {
	WriteGlucose(ctx context.Context, tr *dexcom.TransformedReading) (bool, error)
	ReadGlucose(ctx context.Context, start, end time.Time) ([]dexcom.TransformedReading, error)
}

type MongoStore struct {
	Client *mongo.Client
	Logger *zap.Logger
}

func (ms *MongoStore) writeEvent(ctx context.Context, collection string, event TimePoint) (bool, error) {
	ms.Logger.Debug("inserting event",
		zap.String("collection", collection),
		zap.Any("event", event))

	res, err := ms.Client.
		Database(dbName).
		Collection(collection).
		UpdateOne(ctx, bson.M{
			"time": event.GetTime(),
		}, bson.M{"$set": event}, options.Update().SetUpsert(true))

	if err != nil {
		ms.Logger.Debug("failed to insert event",
			zap.String("collection", collection),
			zap.Any("event", event),
			zap.Error(err))
	}

	return (res.MatchedCount > 0), err
}

func (ms *MongoStore) getEventBetween(ctx context.Context, collection string, start, end time.Time, slicePtr interface{}) error {
	ms.Logger.Debug("reading events",
		zap.String("collection", collection),
		zap.Time("start", start),
		zap.Time("end", end),
	)

	cur, err := ms.Client.
		Database(dbName).
		Collection(collection).
		Find(ctx, bson.M{
			"time": bson.M{
				"$gte": primitive.NewDateTimeFromTime(start),
				"$lte": primitive.NewDateTimeFromTime(end),
			},
		})
	if err != nil {
		ms.Logger.Debug("failed to insert event",
			zap.String("collection", collection),
			zap.Time("start", start),
			zap.Time("end", end),
			zap.Error(err))
		return err
	}

	return cur.All(ctx, slicePtr)
}

func (ms *MongoStore) WriteGlucose(ctx context.Context, tr *dexcom.TransformedReading) (bool, error) {
	return ms.writeEvent(ctx, glucoseCollection, tr)
}

func (ms *MongoStore) ReadGlucose(ctx context.Context, start, end time.Time) ([]dexcom.TransformedReading, error) {
	var trs []dexcom.TransformedReading
	if err := ms.getEventBetween(ctx, glucoseCollection, start, end, &trs); err != nil {
		return nil, err
	}
	return trs, nil
}
