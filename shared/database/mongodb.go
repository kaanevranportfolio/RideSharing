package database

import (
	"context"
	"fmt"
	"time"

	"github.com/rideshare-platform/shared/config"
	"github.com/rideshare-platform/shared/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongoDB represents a MongoDB database connection
type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
	config   *config.DatabaseConfig
	logger   *logger.Logger
}

// NewMongoDB creates a new MongoDB database connection
func NewMongoDB(cfg *config.DatabaseConfig, log *logger.Logger) (*MongoDB, error) {
	// Build connection URI
	var uri string
	if cfg.Username != "" && cfg.Password != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?authSource=admin",
			cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	} else {
		uri = fmt.Sprintf("mongodb://%s:%d/%s",
			cfg.Host, cfg.Port, cfg.Database)
	}

	// Set client options
	clientOptions := options.Client().ApplyURI(uri)

	// Configure connection pool
	clientOptions.SetMaxPoolSize(uint64(cfg.MaxOpenConns))
	clientOptions.SetMinPoolSize(uint64(cfg.MaxIdleConns))
	clientOptions.SetMaxConnIdleTime(time.Duration(cfg.ConnMaxIdleTime) * time.Second)
	clientOptions.SetConnectTimeout(10 * time.Second)
	clientOptions.SetServerSelectionTimeout(5 * time.Second)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Test the connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		client.Disconnect(ctx)
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(cfg.Database)

	log.WithFields(logger.Fields{
		"host":     cfg.Host,
		"port":     cfg.Port,
		"database": cfg.Database,
	}).Info("Connected to MongoDB database")

	return &MongoDB{
		Client:   client,
		Database: database,
		config:   cfg,
		logger:   log,
	}, nil
}

// Close closes the MongoDB connection
func (m *MongoDB) Close(ctx context.Context) error {
	if m.Client != nil {
		m.logger.Logger.Info("Closing MongoDB database connection")
		return m.Client.Disconnect(ctx)
	}
	return nil
}

// Health checks the MongoDB health
func (m *MongoDB) Health(ctx context.Context) error {
	return m.Client.Ping(ctx, readpref.Primary())
}

// Collection returns a collection handle
func (m *MongoDB) Collection(name string) *mongo.Collection {
	return m.Database.Collection(name)
}

// WithTransaction executes a function within a MongoDB transaction
func (m *MongoDB) WithTransaction(ctx context.Context, fn func(mongo.SessionContext) error) error {
	session, err := m.Client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	m.logger.WithContext(ctx).Debug("MongoDB transaction started")

	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		return nil, fn(sc)
	})

	if err != nil {
		m.logger.WithContext(ctx).WithError(err).Error("MongoDB transaction failed")
		return err
	}

	m.logger.WithContext(ctx).Debug("MongoDB transaction completed")
	return nil
}

// MongoRepository provides common MongoDB operations
type MongoRepository struct {
	collection *mongo.Collection
	logger     *logger.Logger
}

// NewMongoRepository creates a new MongoDB repository
func NewMongoRepository(db *MongoDB, collectionName string, logger *logger.Logger) *MongoRepository {
	return &MongoRepository{
		collection: db.Collection(collectionName),
		logger:     logger,
	}
}

// InsertOne inserts a single document
func (r *MongoRepository) InsertOne(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
	start := time.Now()
	result, err := r.collection.InsertOne(ctx, document)
	duration := time.Since(start)

	r.logger.LogDatabaseQuery(ctx, "InsertOne", duration, err)
	return result, err
}

// InsertMany inserts multiple documents
func (r *MongoRepository) InsertMany(ctx context.Context, documents []interface{}) (*mongo.InsertManyResult, error) {
	start := time.Now()
	result, err := r.collection.InsertMany(ctx, documents)
	duration := time.Since(start)

	r.logger.LogDatabaseQuery(ctx, "InsertMany", duration, err)
	return result, err
}

// FindOne finds a single document
func (r *MongoRepository) FindOne(ctx context.Context, filter interface{}) *mongo.SingleResult {
	start := time.Now()
	result := r.collection.FindOne(ctx, filter)
	duration := time.Since(start)

	r.logger.LogDatabaseQuery(ctx, "FindOne", duration, nil)
	return result
}

// Find finds multiple documents
func (r *MongoRepository) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	start := time.Now()
	cursor, err := r.collection.Find(ctx, filter, opts...)
	duration := time.Since(start)

	r.logger.LogDatabaseQuery(ctx, "Find", duration, err)
	return cursor, err
}

// UpdateOne updates a single document
func (r *MongoRepository) UpdateOne(ctx context.Context, filter, update interface{}) (*mongo.UpdateResult, error) {
	start := time.Now()
	result, err := r.collection.UpdateOne(ctx, filter, update)
	duration := time.Since(start)

	r.logger.LogDatabaseQuery(ctx, "UpdateOne", duration, err)
	return result, err
}

// UpdateMany updates multiple documents
func (r *MongoRepository) UpdateMany(ctx context.Context, filter, update interface{}) (*mongo.UpdateResult, error) {
	start := time.Now()
	result, err := r.collection.UpdateMany(ctx, filter, update)
	duration := time.Since(start)

	r.logger.LogDatabaseQuery(ctx, "UpdateMany", duration, err)
	return result, err
}

// ReplaceOne replaces a single document
func (r *MongoRepository) ReplaceOne(ctx context.Context, filter, replacement interface{}) (*mongo.UpdateResult, error) {
	start := time.Now()
	result, err := r.collection.ReplaceOne(ctx, filter, replacement)
	duration := time.Since(start)

	r.logger.LogDatabaseQuery(ctx, "ReplaceOne", duration, err)
	return result, err
}

// DeleteOne deletes a single document
func (r *MongoRepository) DeleteOne(ctx context.Context, filter interface{}) (*mongo.DeleteResult, error) {
	start := time.Now()
	result, err := r.collection.DeleteOne(ctx, filter)
	duration := time.Since(start)

	r.logger.LogDatabaseQuery(ctx, "DeleteOne", duration, err)
	return result, err
}

// DeleteMany deletes multiple documents
func (r *MongoRepository) DeleteMany(ctx context.Context, filter interface{}) (*mongo.DeleteResult, error) {
	start := time.Now()
	result, err := r.collection.DeleteMany(ctx, filter)
	duration := time.Since(start)

	r.logger.LogDatabaseQuery(ctx, "DeleteMany", duration, err)
	return result, err
}

// CountDocuments counts documents matching a filter
func (r *MongoRepository) CountDocuments(ctx context.Context, filter interface{}) (int64, error) {
	start := time.Now()
	count, err := r.collection.CountDocuments(ctx, filter)
	duration := time.Since(start)

	r.logger.LogDatabaseQuery(ctx, "CountDocuments", duration, err)
	return count, err
}

// Aggregate performs an aggregation operation
func (r *MongoRepository) Aggregate(ctx context.Context, pipeline interface{}) (*mongo.Cursor, error) {
	start := time.Now()
	cursor, err := r.collection.Aggregate(ctx, pipeline)
	duration := time.Since(start)

	r.logger.LogDatabaseQuery(ctx, "Aggregate", duration, err)
	return cursor, err
}

// CreateIndex creates an index on the collection
func (r *MongoRepository) CreateIndex(ctx context.Context, model mongo.IndexModel) (string, error) {
	start := time.Now()
	name, err := r.collection.Indexes().CreateOne(ctx, model)
	duration := time.Since(start)

	r.logger.LogDatabaseQuery(ctx, "CreateIndex", duration, err)
	return name, err
}

// CreateIndexes creates multiple indexes on the collection
func (r *MongoRepository) CreateIndexes(ctx context.Context, models []mongo.IndexModel) ([]string, error) {
	start := time.Now()
	names, err := r.collection.Indexes().CreateMany(ctx, models)
	duration := time.Since(start)

	r.logger.LogDatabaseQuery(ctx, "CreateIndexes", duration, err)
	return names, err
}

// DropIndex drops an index from the collection
func (r *MongoRepository) DropIndex(ctx context.Context, name string) error {
	start := time.Now()
	_, err := r.collection.Indexes().DropOne(ctx, name)
	duration := time.Since(start)

	r.logger.LogDatabaseQuery(ctx, "DropIndex", duration, err)
	return err
}

// ListIndexes lists all indexes on the collection
func (r *MongoRepository) ListIndexes(ctx context.Context) (*mongo.Cursor, error) {
	start := time.Now()
	cursor, err := r.collection.Indexes().List(ctx)
	duration := time.Since(start)

	r.logger.LogDatabaseQuery(ctx, "ListIndexes", duration, err)
	return cursor, err
}

// BulkWrite performs multiple write operations
func (r *MongoRepository) BulkWrite(ctx context.Context, models []mongo.WriteModel) (*mongo.BulkWriteResult, error) {
	start := time.Now()
	result, err := r.collection.BulkWrite(ctx, models)
	duration := time.Since(start)

	r.logger.LogDatabaseQuery(ctx, "BulkWrite", duration, err)
	return result, err
}

// Watch creates a change stream for the collection
func (r *MongoRepository) Watch(ctx context.Context, pipeline interface{}) (*mongo.ChangeStream, error) {
	start := time.Now()
	stream, err := r.collection.Watch(ctx, pipeline)
	duration := time.Since(start)

	r.logger.LogDatabaseQuery(ctx, "Watch", duration, err)
	return stream, err
}

// Distinct gets distinct values for a field
func (r *MongoRepository) Distinct(ctx context.Context, fieldName string, filter interface{}) ([]interface{}, error) {
	start := time.Now()
	values, err := r.collection.Distinct(ctx, fieldName, filter)
	duration := time.Since(start)

	r.logger.LogDatabaseQuery(ctx, "Distinct", duration, err)
	return values, err
}

// EstimatedDocumentCount gets an estimated count of documents
func (r *MongoRepository) EstimatedDocumentCount(ctx context.Context) (int64, error) {
	start := time.Now()
	count, err := r.collection.EstimatedDocumentCount(ctx)
	duration := time.Since(start)

	r.logger.LogDatabaseQuery(ctx, "EstimatedDocumentCount", duration, err)
	return count, err
}
