package store

import (
	"bytes"
	"context"
	"io"
	"iv2/gourgeist/dexcom"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const (
	glucoseCollection = "glucose"
	filesCollection   = "fs.files"
)

type TimePoint interface {
	GetTime() time.Time
}

type Store interface {
	WriteGlucose(ctx context.Context, tr *dexcom.TransformedReading) (bool, error)
	ReadGlucose(ctx context.Context, start, end time.Time) ([]dexcom.TransformedReading, error)

	ReadFile(ctx context.Context, fid string) (io.Reader, error)
	DeleteFile(ctx context.Context, fid string) error
}

type MongoStore struct {
	Client *mongo.Client
	Logger *zap.Logger

	DBName string
}

func New(ctx context.Context, uri, dbName string, logger *zap.Logger) (Store, error) {
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	return &MongoStore{
		Client: mongoClient,
		Logger: logger,
		DBName: dbName,
	}, nil
}

func (ms *MongoStore) writeEvent(ctx context.Context, collection string, event TimePoint) (bool, error) {
	ms.Logger.Debug("inserting event",
		zap.String("collection", collection),
		zap.Any("event", event))

	res, err := ms.Client.
		Database(ms.DBName).
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
		Database(ms.DBName).
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

func (ms *MongoStore) ReadFile(ctx context.Context, fid string) (io.Reader, error) {
	db := ms.Client.Database(ms.DBName)
	bucket, err := gridfs.NewBucket(db)
	if err != nil {
		return nil, err
	}

	oid, err := primitive.ObjectIDFromHex(fid)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	_, err = bucket.DownloadToStream(oid, &buf)
	if err != nil {
		return nil, err
	}

	return &buf, nil
}

func (ms *MongoStore) DeleteFile(ctx context.Context, fid string) error {
	db := ms.Client.Database(ms.DBName)
	bucket, err := gridfs.NewBucket(db)
	if err != nil {
		return err
	}

	oid, err := primitive.ObjectIDFromHex(fid)
	if err != nil {
		return err
	}

	return bucket.Delete(oid)
}
