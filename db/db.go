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

	collection := client.Database("crawlerArchive").Collection("webpages")
    if err := collection.Drop(context.Background()); err != nil {
        log.Printf("Warning: Failed to drop collection: %v", err)
    } else {
        log.Println("Dropped webpages collection on restart")
    }
	
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

// func (db *DB) GetCollection(database, collection string) *mongo.Collection {
// 	return db.Client.Database(database).Collection(collection)
// }

func (db *DB) InsertWebpage(collection string, data interface{}) error {
    coll := db.Client.Database("crawlerArchive").Collection(collection)
    _, err := coll.InsertOne(context.TODO(), data)
    if err != nil {
        log.Printf("Failed to insert into %s: %v", collection, err)
        return err
    }
    return nil
}