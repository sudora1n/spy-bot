package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Migration struct {
	Name      string    `bson:"name"`
	AppliedAt time.Time `bson:"applied_at"`
}

func (r *MongoRepository) MigrationIsNeeded(ctx context.Context, name string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var result Migration
	err := r.migrations.FindOne(ctx, bson.M{"name": name}).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return false, nil
}

func (r *MongoRepository) ApplyMigration(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.migrations.InsertOne(ctx, Migration{
		Name:      name,
		AppliedAt: time.Now(),
	})
	return err
}
