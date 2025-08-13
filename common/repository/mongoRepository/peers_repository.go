package mongoRepository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Peer struct {
	ID        int64  `bson:"_id"`
	FirstName string `bson:"first_name"`
	LastName  string `bson:"last_name,omitempty"`
	Username  string `bson:"username,omitempty"`
}

func (r *MongoRepository) GetPeer(ctx context.Context, peerID int64) (*Peer, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var peer Peer
	err := r.peers.FindOne(ctx, bson.M{"_id": peerID}).Decode(&peer)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("peer not found")
		}
		return nil, fmt.Errorf("failed to find peer: %w", err)
	}

	return &peer, nil
}

func (r *MongoRepository) syncPeer(ctx context.Context, peer *Peer) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"first_name": peer.FirstName,
			"last_name":  peer.LastName,
			"username":   peer.Username,
		},
	}

	_, err := r.peers.UpdateOne(
		ctx,
		bson.M{"_id": peer.ID},
		update,
		options.Update().SetUpsert(true),
	)

	return err
}
