package mg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"iv2/gourgeist/defs"
	"reflect"
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

type MongoStore struct {
	Client *mongo.Client
	Logger *zap.Logger

	Database *mongo.Database
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

	err = mongoClient.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to ping mongo: %w", err)
	}

	return &MongoStore{
		Client:   mongoClient,
		Logger:   logger,
		Database: mongoClient.Database(dbName),
	}, nil
}

type DocumentStore interface {
	DocByID(ctx context.Context, collection, id string, doc interface{}) error
	DeleteByID(ctx context.Context, collection string, id string) error
	InsertNew(ctx context.Context, collection string, doc interface{}) (*defs.UpdateResult, error)
	Update(ctx context.Context, collection string, id string, doc interface{}) (*defs.UpdateResult, error)
}

func (ms *MongoStore) DocByID(ctx context.Context, collection, id string, doc interface{}) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	sr := ms.Database.Collection(collection).FindOne(ctx, bson.M{"_id": oid})
	return sr.Decode(doc)
}

func (ms *MongoStore) DeleteByID(ctx context.Context, collection string, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = ms.Database.Collection(collection).DeleteOne(ctx, bson.M{"_id": oid})
	return err
}

func (ms *MongoStore) InsertNew(ctx context.Context, collection string, doc interface{}) (*defs.UpdateResult, error) {
	ms.Logger.Debug(
		"inserting document",
		zap.String("collection", collection),
		zap.Any("document", doc),
	)

	filter := bson.M{}

	r := reflect.ValueOf(doc)
	f := reflect.Indirect(r).FieldByName("Time")
	fieldValue := f.Interface()

	switch v := fieldValue.(type) {
	case time.Time:
		filter["time"] = v
	}

	res, err := ms.Database.
		Collection(collection).
		UpdateOne(ctx, filter,
			bson.M{"$setOnInsert": doc},
			options.Update().SetUpsert(true),
		)
	if err != nil {
		return nil, fmt.Errorf("unable to insert if new: %w", err)
	}

	oid, _ := res.UpsertedID.(primitive.ObjectID)
	return &defs.UpdateResult{
		MatchedCount:  res.MatchedCount,
		ModifiedCount: res.ModifiedCount,
		UpsertedCount: res.UpsertedCount,
		UpsertedID:    defs.MyObjectID(oid.Hex()),
	}, nil
}

func (ms *MongoStore) Update(ctx context.Context, collection string, id string, doc interface{}) (*defs.UpdateResult, error) {
	ms.Logger.Debug(
		"updating document",
		zap.String("collection", collection),
		zap.Any("document", doc),
	)

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	res, err := ms.Database.
		Collection(collection).
		UpdateOne(ctx,
			bson.M{"_id": oid},
			bson.M{"$set": doc},
			options.Update().SetUpsert(true),
		)
	if err != nil {
		return nil, fmt.Errorf("unable to update document: %w", err)
	}

	return &defs.UpdateResult{
		MatchedCount:  res.MatchedCount,
		ModifiedCount: res.ModifiedCount,
		UpsertedCount: res.UpsertedCount,
		UpsertedID:    defs.MyObjectID(oid.Hex()),
	}, nil
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

	cur, err := ms.Database.
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

type GlucoseStore interface {
	WriteGlucose(ctx context.Context, tr *defs.TransformedReading) (*defs.UpdateResult, error)
	ReadGlucose(ctx context.Context, start, end time.Time) ([]defs.TransformedReading, error)
}

func (ms *MongoStore) WriteGlucose(ctx context.Context, tr *defs.TransformedReading) (*defs.UpdateResult, error) {
	return ms.InsertNew(ctx, GlucoseCollection, tr)
}

func (ms *MongoStore) ReadGlucose(ctx context.Context, start, end time.Time) ([]defs.TransformedReading, error) {
	var trs []defs.TransformedReading
	if err := ms.getEventsBetween(ctx, GlucoseCollection, start, end, &trs); err != nil {
		return nil, fmt.Errorf("unable to read glucose: %w", err)
	}
	return trs, nil
}

type InsulinStore interface {
	WriteInsulin(ctx context.Context, in *defs.Insulin) (*defs.UpdateResult, error)
	UpdateInsulin(ctx context.Context, in *defs.Insulin) (*defs.UpdateResult, error)
	ReadInsulin(ctx context.Context, start, end time.Time) ([]defs.Insulin, error)
}

func (ms *MongoStore) WriteInsulin(ctx context.Context, in *defs.Insulin) (*defs.UpdateResult, error) {
	return ms.InsertNew(ctx, InsulinCollection, in)
}

func (ms *MongoStore) UpdateInsulin(ctx context.Context, in *defs.Insulin) (*defs.UpdateResult, error) {
	return ms.Update(ctx, InsulinCollection, string(in.ID), in)
}

func (ms *MongoStore) ReadInsulin(ctx context.Context, start, end time.Time) ([]defs.Insulin, error) {
	var ins []defs.Insulin
	if err := ms.getEventsBetween(ctx, InsulinCollection, start, end, &ins); err != nil {
		return nil, fmt.Errorf("unable to read insulin: %w", err)
	}
	return ins, nil
}

type CarbStore interface {
	WriteCarbs(ctx context.Context, c *defs.Carb) (*defs.UpdateResult, error)
	UpdateCarbs(ctx context.Context, c *defs.Carb) (*defs.UpdateResult, error)
	ReadCarbs(ctx context.Context, start, end time.Time) ([]defs.Carb, error)
}

func (ms *MongoStore) WriteCarbs(ctx context.Context, c *defs.Carb) (*defs.UpdateResult, error) {
	return ms.InsertNew(ctx, CarbsCollection, c)
}

func (ms *MongoStore) UpdateCarbs(ctx context.Context, c *defs.Carb) (*defs.UpdateResult, error) {
	return ms.Update(ctx, CarbsCollection, string(c.ID), c)
}

func (ms *MongoStore) ReadCarbs(ctx context.Context, start, end time.Time) ([]defs.Carb, error) {
	var carbs []defs.Carb
	if err := ms.getEventsBetween(ctx, CarbsCollection, start, end, &carbs); err != nil {
		return nil, fmt.Errorf("unable to read carbs: %w", err)
	}
	return carbs, nil
}

type AlertStore interface {
	WriteAlert(ctx context.Context, al *defs.Alert) (*defs.UpdateResult, error)
	ReadAlerts(ctx context.Context, start, end time.Time) ([]defs.Alert, error)
}

func (ms *MongoStore) WriteAlert(ctx context.Context, al *defs.Alert) (*defs.UpdateResult, error) {
	return ms.InsertNew(ctx, AlertsCollection, al)
}

func (ms *MongoStore) ReadAlerts(ctx context.Context, start, end time.Time) ([]defs.Alert, error) {
	var alerts []defs.Alert
	if err := ms.getEventsBetween(ctx, AlertsCollection, start, end, &alerts); err != nil {
		return nil, fmt.Errorf("unable to read alerts: %w", err)
	}
	return alerts, nil
}

type FileStore interface {
	ReadFile(ctx context.Context, fid string) (io.Reader, error)
	DeleteFile(ctx context.Context, fid string) error
}

func (ms *MongoStore) ReadFile(ctx context.Context, fid string) (io.Reader, error) {
	bucket, err := gridfs.NewBucket(ms.Database)
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
	bucket, err := gridfs.NewBucket(ms.Database)
	if err != nil {
		return fmt.Errorf("unable to create a GridFS bucket: %w", err)
	}

	oid, err := primitive.ObjectIDFromHex(fid)
	if err != nil {
		return fmt.Errorf("unable to create objectId from hex: %w", err)
	}

	return bucket.Delete(oid)
}
