package database

import (
	"context"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	client		*mongo.Client
	database	*mongo.Database
	once 		sync.Once
)

// Connect establishes the MongoDB connection using the provided URI.
func Connect(uri string) error{
	var connectErr error
	once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		c, err := mongo.Connect(options.Client().ApplyURI(uri))
		if err != nil {
			connectErr = err
			return
		}

		if err := c.Ping(ctx, nil); err != nil {
			connectErr = err
			return
		}

		client = c
		database = c.Database("devflow")
		log.Println("Connected to MongoDB!")
	})
	return connectErr
}

// GetDB returns the devflow database handle.
func GetDB() *mongo.Database {
	return database
}

// GetClient returns the raw mongo client.
func GetClient() *mongo.Client {
	return client
}

// Disconnect closes the MongoDB connection.
func Disconnect() {
	if client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := client.Disconnect(ctx); err != nil {
			log.Printf("MongoDB disconnect error: %v", err)
		} else {
			log.Println("MongoDB disconnected")
		}
	}
}

// Collection returns a handle to the named collection.
func Collection(name string) *mongo.Collection {
	return database.Collection(name)
}