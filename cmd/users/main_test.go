package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func TestForExpectedGreeting(t *testing.T) {
	require.Equal(t, "Hello, World", Greeting())
}

func TestCanConnectToMongodbService(t *testing.T) {
	// Copy pasted from the mongodb go client docs. Test connectivity to the db
	uri := "mongodb://root:password@localhost:27017/test?maxPoolSize=20&w=majority&authSource=admin"
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}
	defer func() {
		if err = client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()
	// Ping the primary
	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		panic(err)
	}
}
