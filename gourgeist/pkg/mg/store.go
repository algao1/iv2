package mg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"iv2/gourgeist/defs"
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
	AlertsCollection  = "alerts"
	FilesCollection   = "fs.files"
)

type DocumentStore interface {
	DocByID(ctx context.Context, collection string, id *primitive.ObjectID, doc interface{}) error
	DeleteByID(ctx context.Context, collection string, id *primitive.ObjectID) error
	InsertIfNew(ctx context.Context, collection string, filter bson.M, doc interface{}) (*mongo.UpdateResult, error)
	Upsert(ctx context.Context, collection string, filter bson.M, doc interface{}) (*mongo.UpdateResult, error)
}

type GlucoseStore interface {
	WriteGlucose(ctx context.Context, tr *defs.TransformedReading) (*mongo.UpdateResult, error)
	ReadGlucose(ctx context.Context, start, end time.Time) ([]defs.TransformedReading, error)
}

type InsulinStore interface {
	WriteInsulin(ctx context.Context, in *defs.Insulin) (*mongo.UpdateResult, error)
	ReadInsulin(ctx context.Context, start, end time.Time) ([]defs.Insulin, error)
}

type CarbStore interface {
	WriteCarbs(ctx context.Context, c *defs.Carb) (*mongo.UpdateResult, error)
	ReadCarbs(ctx context.Context, start, end time.Time) ([]defs.Carb, error)
}

type AlertStore interface {
	WriteAlert(ctx context.Context, al *defs.Alert) (*mongo.UpdateResult, error)
	ReadAlerts(ctx context.Context, start, end time.Time) ([]defs.Alert, error)
}

type FileStore interface {
	ReadFile(ctx context.Context, fid string) (io.Reader, error)
	DeleteFile(ctx context.Context, fid string) error
}

type MongoStore struct {
	Client *mongo.Client
	Logger *zap.Logger

	DBName string
}

func New(ctx context.Context, cfg defs.MongoConfig, dbName string, logger *zap.Logger) (*MongoStore, error) {
	mongoClient, err := mongo.Connect(
		ctx,
		options.Client().ApplyURI(cfg.URI),
		options.Client().SetAuth(options.Credential{
			Username: cfg.Username,
			Password: cfg.Password,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to mongo: %w", err)
	}

	return &MongoStore{
		Client: mongoClient,
		Logger: logger,
		DBName: dbName,
	}, nil
}

func (ms *MongoStore) DocByID(ctx context.Context, collection string, id *primitive.ObjectID, doc interface{}) error {
	sr := ms.Client.Database(ms.DBName).Collection(collection).FindOne(ctx, bson.M{"_id": id})
	return sr.Decode(doc)
}

func (ms *MongoStore) InsertIfNew(ctx context.Context, collection string, filter bson.M, doc interface{}) (*mongo.UpdateResult, error) {
	ms.Logger.Debug(
		"inserting document",
		zap.String("collection", collection),
		zap.Any("filter", filter),
		zap.Any("document", doc),
	)

	res, err := ms.Client.
		Database(ms.DBName).
		Collection(collection).
		UpdateOne(ctx, filter,
			bson.M{"$setOnInsert": doc},
			options.Update().SetUpsert(true),
		)
	if err != nil {
		return nil, fmt.Errorf("unable to insert if new: %w", err)
	}

	return res, err
}

func (ms *MongoStore) Upsert(ctx context.Context, collection string, filter bson.M, doc interface{}) (*mongo.UpdateResult, error) {
	ms.Logger.Debug(
		"upeserting document",
		zap.String("collection", collection),
		zap.Any("document", doc),
	)

	res, err := ms.Client.
		Database(ms.DBName).
		Collection(collection).
		UpdateOne(ctx, filter,
			bson.M{"$set": doc},
			options.Update().SetUpsert(true),
		)
	if err != nil {
		ms.Logger.Debug(
			"unable to upsert document",
			zap.String("collection", collection),
			zap.Any("document", doc),
			zap.Error(err),
		)
		return nil, fmt.Errorf("unable to upsert document: %w", err)
	}

	return res, err
}

func (ms *MongoStore) DeleteByID(ctx context.Context, collection string, id *primitive.ObjectID) error {
	ms.Logger.Debug(
		"deleting document by id",
		zap.String("collection", collection),
		zap.String("id", id.Hex()),
	)
	_, err := ms.Client.Database(ms.DBName).Collection(collection).DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (ms *MongoStore) getEventsBetween(ctx context.Context, collection string, start, end time.Time, slicePtr interface{}) error {
	ms.Logger.Debug(
		"reading events",
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
		ms.Logger.Debug(
			"unable to read events",
			zap.String("collection", collection),
			zap.Time("start", start),
			zap.Time("end", end),
			zap.Error(err),
		)
		return fmt.Errorf("unable to read events: %w", err)
	}

	return cur.All(ctx, slicePtr)
}

func (ms *MongoStore) WriteGlucose(ctx context.Context, tr *defs.TransformedReading) (*mongo.UpdateResult, error) {
	filter := bson.M{"time": tr.Time}
	return ms.InsertIfNew(ctx, GlucoseCollection, filter, tr)
}

func (ms *MongoStore) ReadGlucose(ctx context.Context, start, end time.Time) ([]defs.TransformedReading, error) {
	var trs []defs.TransformedReading
	if err := ms.getEventsBetween(ctx, GlucoseCollection, start, end, &trs); err != nil {
		return nil, fmt.Errorf("unable to read glucose: %w", err)
	}
	return trs, nil
}

func (ms *MongoStore) WriteInsulin(ctx context.Context, in *defs.Insulin) (*mongo.UpdateResult, error) {
	filter := bson.M{}
	if in.ID != nil {
		filter["_id"] = in.ID
	} else {
		filter["time"] = in.Time
	}
	return ms.Upsert(ctx, InsulinCollection, filter, in)
}

func (ms *MongoStore) ReadInsulin(ctx context.Context, start, end time.Time) ([]defs.Insulin, error) {
	var ins []defs.Insulin
	if err := ms.getEventsBetween(ctx, InsulinCollection, start, end, &ins); err != nil {
		return nil, fmt.Errorf("unable to read insulin: %w", err)
	}
	return ins, nil
}

func (ms *MongoStore) WriteCarbs(ctx context.Context, c *defs.Carb) (*mongo.UpdateResult, error) {
	filter := bson.M{}
	if c.ID != nil {
		filter["_id"] = c.ID
	} else {
		filter["time"] = c.Time
	}
	return ms.Upsert(ctx, CarbsCollection, filter, c)
}

func (ms *MongoStore) ReadCarbs(ctx context.Context, start, end time.Time) ([]defs.Carb, error) {
	var carbs []defs.Carb
	if err := ms.getEventsBetween(ctx, CarbsCollection, start, end, &carbs); err != nil {
		return nil, fmt.Errorf("unable to read carbs: %w", err)
	}
	return carbs, nil
}

func (ms *MongoStore) WriteAlert(ctx context.Context, al *defs.Alert) (*mongo.UpdateResult, error) {
	filter := bson.M{}
	if al.ID != nil {
		filter["_id"] = al.ID
	} else {
		filter["time"] = al.Time
	}
	return ms.Upsert(ctx, AlertsCollection, filter, al)
}

func (ms *MongoStore) ReadAlerts(ctx context.Context, start, end time.Time) ([]defs.Alert, error) {
	var alerts []defs.Alert
	if err := ms.getEventsBetween(ctx, AlertsCollection, start, end, &alerts); err != nil {
		return nil, fmt.Errorf("unable to read alerts: %w", err)
	}
	return alerts, nil
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
