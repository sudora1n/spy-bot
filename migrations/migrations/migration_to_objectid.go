package migrations

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func MigrateCollectionToObjectID(ctx context.Context, db *mongo.Database, collectionName string) error {
	tempCollectionName := collectionName + "_new"

	oldCollection := db.Collection(collectionName)

	newCollection := db.Collection(tempCollectionName)

	cursor, err := oldCollection.Find(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("error while getting docs: %v", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return fmt.Errorf("error decode: %v", err)
		}

		delete(doc, "_id")

		_, err := newCollection.InsertOne(ctx, doc)
		if err != nil {
			return fmt.Errorf("error insert doc: %v", err)
		}
	}

	oldCount, err := oldCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("error get len of old coll: %v", err)
	}

	newCount, err := newCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("error get len of new coll: %v", err)
	}

	if oldCount != newCount {
		return fmt.Errorf("len doesnt correct: old=%d, new=%d", oldCount, newCount)
	}

	if err := oldCollection.Drop(ctx); err != nil {
		return fmt.Errorf("error delete old coll: %v", err)
	}

	renameCmd := bson.D{
		{Key: "renameCollection", Value: db.Name() + "." + tempCollectionName},
		{Key: "to", Value: db.Name() + "." + collectionName},
	}
	if err := db.RunCommand(ctx, renameCmd).Err(); err != nil {
		return fmt.Errorf("error rename new coll: %v", err)
	}

	return nil
}
