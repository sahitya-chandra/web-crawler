package db

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DB struct {
	client   *mongo.Client
	database string
}

func Connect(ctx context.Context, uri, database string) (*DB, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("mongo ping: %w", err)
	}

	log.Println("Connected to MongoDB")

	return &DB{
		client:   client,
		database: database,
	}, nil
}

func (db *DB) Disconnect(ctx context.Context) error {
	if db.client == nil {
		return nil
	}

	if err := db.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("mongo disconnect: %w", err)
	}

	log.Println("Disconnected from MongoDB")
	return nil
}

type Webpage struct {
	URL     string `bson:"url"`
	Title   string `bson:"title"`
	Content string `bson:"content"`
}

func (db *DB) InsertWebpage(ctx context.Context, collection string, page Webpage) error {
	coll := db.client.Database(db.database).Collection(collection)
	_, err := coll.InsertOne(ctx, page)
	if err != nil {
		return fmt.Errorf("insert into %s: %w", collection, err)
	}
	return nil
}
