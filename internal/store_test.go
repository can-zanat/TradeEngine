package internal

import (
	"context"
	"log"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	mongoImage = "mongo:7.0.4"
)

func NewStoreWithURI(uri string) *MongoDBStore {
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(context.Background(), clientOptions)

	if err != nil {
		log.Fatal("Connection failure:", err)
	}

	if err = client.Ping(context.Background(), nil); err != nil {
		log.Fatal("Unable to access MongoDB:", err)
	}

	return &MongoDBStore{
		Client: client,
	}
}

func prepareTestStore(t *testing.T) (store *MongoDBStore, clean func()) {
	t.Helper()

	ctx := context.Background()

	mongodbContainer, err := mongodb.RunContainer(ctx, testcontainers.WithImage(mongoImage))
	if err != nil {
		t.Fatalf("Failed to start MongoDB container: %v", err)
	}

	clean = func() {
		if terminateErr := mongodbContainer.Terminate(ctx); terminateErr != nil {
			t.Fatalf("Failed to terminate MongoDB container: %v", terminateErr)
		}
	}

	containerURI, err := mongodbContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("Failed to get container connection string: %v", err)
	}

	s := NewStoreWithURI(containerURI)

	return s, clean
}

func TestCreateOrderLog(t *testing.T) {
	store, clean := prepareTestStore(t)
	defer clean()

	testOrder := map[string]interface{}{
		"symbol": "BTCUSDT",
		"price":  "50000",
	}

	err := store.CreateOrderLog(testOrder)
	if err != nil {
		t.Fatalf("error creating order log: %v", err)
	}

	collection := store.Client.Database("orders").Collection("orderLog")

	var result bson.M

	if findErr := collection.FindOne(context.Background(), bson.M{"symbol": "BTCUSDT"}).Decode(&result); findErr != nil {
		t.Fatalf("order not found in DB: %v", findErr)
	}

	if result["price"] != "50000" {
		t.Errorf("expected price to be '50000', got %v", result["price"])
	}
}
