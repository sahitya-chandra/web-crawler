package db

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DB struct {
	Client *mongo.Client
}

func Connect(uri string) (*DB, error) {
	
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		return nil, err
	}

	log.Println("Connected to MongoDB!")
	return &DB {
		Client: client,
	}, nil
}

func (db *DB) Disconnect() error {
	if db.Client == nil {
		return nil
	}

	err := db.Client.Disconnect(context.TODO())
	if err != nil {
		return err
	}

	log.Println("Disconnected from MongoDB!")
	return nil
}

func (db *DB) GetCollection(database, collection string) *mongo.Collection {
	return db.Client.Database(database).Collection(collection)
}