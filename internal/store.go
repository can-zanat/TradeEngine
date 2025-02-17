package internal

import (
	"TradeEngine/configs"
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	orderLogSuccessCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "order_log_success_total",
		Help: "Number of successfully logged orders",
	})
	orderLogFailureCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "order_log_failure_total",
		Help: "Number of failed order logs",
	})
)

type Store interface {
	ListenRatesData(callback func(float64, string)) error
	CreateOrderLog(order map[string]interface{}) error
}

type MongoDBStore struct {
	Client *mongo.Client
}

func NewStore() *MongoDBStore {
	config, err := configs.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	clientOptions := options.Client().ApplyURI(config.MongoDB.URI)

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

func (store *MongoDBStore) ListenRatesData(callback func(float64, string)) error {
	ctx := context.Background()
	collection := store.Client.Database("rates").Collection("rateBTC/USDT")

	pipeline := mongo.Pipeline{}

	changeStream, err := collection.Watch(ctx, pipeline)
	if err != nil {
		return fmt.Errorf("error starting change stream: %v", err)
	}
	defer changeStream.Close(ctx)

	log.Println("Started watching changes on rateBTC/USDT collection...")

	for changeStream.Next(ctx) {
		var changeDoc bson.M
		if err := changeStream.Decode(&changeDoc); err != nil {
			log.Printf("Error decoding change stream document: %v", err)
			continue
		}

		fullDoc, ok := changeDoc["fullDocument"].(bson.M)
		if !ok {
			log.Println("fullDocument not found in change event")
			continue
		}

		cVal, ok := fullDoc["c"]
		if !ok {
			log.Println("Field 'c' not found in fullDocument")
			continue
		}

		cStr, ok := cVal.(string)
		if !ok {
			log.Println("Field 'c' is not a string")
			continue
		}

		cFloat, err := strconv.ParseFloat(cStr, 64)
		if err != nil {
			log.Printf("Error converting 'c' value to float64: %v", err)
			continue
		}

		sVal, ok := fullDoc["s"]
		if !ok {
			log.Println("Field 's' not found in fullDocument")
			continue
		}

		sStr, ok := sVal.(string)
		if !ok {
			log.Println("Field 's' is not a string")
			continue
		}

		callback(cFloat, sStr)
	}

	if err = changeStream.Err(); err != nil {
		return fmt.Errorf("change stream error: %v", err)
	}

	return nil
}

func (store *MongoDBStore) Close() {
	if err := store.Client.Disconnect(context.Background()); err != nil {
		log.Fatalf("Error disconnecting from MongoDB: %v", err)
	}
}

func (store *MongoDBStore) CreateOrderLog(order map[string]interface{}) error {
	ctx := context.Background()
	collection := store.Client.Database("orders").Collection("orderLog")

	_, err := collection.InsertOne(ctx, order)
	if err != nil {
		orderLogFailureCounter.Inc()
		return fmt.Errorf("error inserting order log: %v", err)
	}

	orderLogSuccessCounter.Inc()

	return nil
}
