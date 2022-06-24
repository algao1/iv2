package mongo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"iv2/gourgeist/types"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const (
	GlucoseCollection = "glucose"
	InsulinCollection = "insulin"
	CarbsCollection   = "carbs"
	FilesCollection   = "fs.files"
)

type Store interface {
	DocById(ctx context.Context, collection string, id *primitive.ObjectID, doc interface{}) error

	WriteGlucose(ctx context.Context, tr *types.TransformedReading) (*mongo.UpdateResult, error)
	ReadGlucose(ctx context.Context, start, end time.Time) ([]types.TransformedReading, error)

	WriteInsulin(ctx context.Context, in *types.Insulin) (*mongo.UpdateResult, error)
	ReadInsulin(ctx context.Context, start, end time.Time) ([]types.Insulin, error)

	WriteCarbs(ctx context.Context, c *types.Carb) (*mongo.UpdateResult, error)
	ReadCarbs(ctx context.Context, start, end time.Time) ([]types.Carb, error)

	ReadFile(ctx context.Context, fid string) (io.Reader, error)
	DeleteFile(ctx context.Context, fid string) error
}

type MongoStore struct {
	Client *mongo.Client
	Logger *zap.Logger

	DBName string
}

func New(ctx context.Context, uri, dbName string, logger *zap.Logger) (*MongoStore, error) {
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("unable to connect to mongo: %w", err)
	}
	return &MongoStore{
		Client: mongoClient,
		Logger: logger,
		DBName: dbName,
	}, nil
}

func (ms *MongoStore) DocById(ctx context.Context, collection string, id *primitive.ObjectID, doc interface{}) error {
	sr := ms.Client.Database(ms.DBName).Collection(collection).FindOne(ctx, bson.M{"_id": id})
	return sr.Decode(doc)
}

func (ms *MongoStore) writeEvent(ctx context.Context, collection string, event types.TimePoint) (*mongo.UpdateResult, error) {
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
		ms.Logger.Debug("unable to insert event",
			zap.String("collection", collection),
			zap.Any("event", event),
			zap.Error(err))
		return nil, fmt.Errorf("unable to insert event: %w", err)
	}

	return res, nil
}

func (ms *MongoStore) getEventBetween(ctx context.Context, collection string, start, end time.Time, slicePtr interface{}) error {
	ms.Logger.Debug("reading events",
		zap.String("collection", collection),
		zap.Time("start", start),
		zap.Time("end", end),
	)

	findOptions := options.Find()
	findOptions.SetSort(bson.D{primitive.E{Key: "time", Value: 1}})

	cur, err := ms.Client.
		Database(ms.DBName).
		Collection(collection).
		Find(ctx, bson.M{
			"time": bson.M{
				"$gte": primitive.NewDateTimeFromTime(start),
				"$lte": primitive.NewDateTimeFromTime(end),
			},
		}, findOptions)
	if err != nil {
		ms.Logger.Debug("unable to read events",
			zap.String("collection", collection),
			zap.Time("start", start),
			zap.Time("end", end),
			zap.Error(err))
		return fmt.Errorf("unable to read events: %w", err)
	}

	return cur.All(ctx, slicePtr)
}

func (ms *MongoStore) WriteGlucose(ctx context.Context, tr *types.TransformedReading) (*mongo.UpdateResult, error) {
	return ms.writeEvent(ctx, GlucoseCollection, tr)
}

func (ms *MongoStore) ReadGlucose(ctx context.Context, start, end time.Time) ([]types.TransformedReading, error) {
	var trs []types.TransformedReading
	if err := ms.getEventBetween(ctx, GlucoseCollection, start, end, &trs); err != nil {
		return nil, fmt.Errorf("unable to read glucose: %w", err)
	}
	return trs, nil
}

func (ms *MongoStore) WriteInsulin(ctx context.Context, in *types.Insulin) (*mongo.UpdateResult, error) {
	return ms.writeEvent(ctx, InsulinCollection, in)
}

func (ms *MongoStore) ReadInsulin(ctx context.Context, start, end time.Time) ([]types.Insulin, error) {
	var ins []types.Insulin
	if err := ms.getEventBetween(ctx, InsulinCollection, start, end, &ins); err != nil {
		return nil, fmt.Errorf("unable to read insulin: %w", err)
	}
	return ins, nil
}

func (ms *MongoStore) WriteCarbs(ctx context.Context, c *types.Carb) (*mongo.UpdateResult, error) {
	return ms.writeEvent(ctx, CarbsCollection, c)
}

func (ms *MongoStore) ReadCarbs(ctx context.Context, start, end time.Time) ([]types.Carb, error) {
	var carbs []types.Carb
	if err := ms.getEventBetween(ctx, CarbsCollection, start, end, &carbs); err != nil {
		return nil, fmt.Errorf("unable to read carbs: %w", err)
	}
	return carbs, nil
}

func (ms *MongoStore) ReadFile(ctx context.Context, fid string) (io.Reader, error) {
	db := ms.Client.Database(ms.DBName)
	bucket, err := gridfs.NewBucket(db)
	if err != nil {
		return nil, fmt.Errorf("unable to create a GridFS bucket: %w", err)
	}

	oid, err := primitive.ObjectIDFromHex(fid)
	if err != nil {
		return nil, fmt.Errorf("unable to create objectId from hex: %w", err)
	}

	var buf bytes.Buffer
	_, err = bucket.DownloadToStream(oid, &buf)
	if err != nil {
		return nil, fmt.Errorf("unable to download to stream: %w", err)
	}

	return &buf, nil
}

func (ms *MongoStore) DeleteFile(ctx context.Context, fid string) error {
	db := ms.Client.Database(ms.DBName)
	bucket, err := gridfs.NewBucket(db)
	if err != nil {
		return fmt.Errorf("unable to create a GridFS bucket: %w", err)
	}

	oid, err := primitive.ObjectIDFromHex(fid)
	if err != nil {
		return fmt.Errorf("unable to create objectId from hex: %w", err)
	}

	return bucket.Delete(oid)
}
