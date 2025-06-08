package types

import (
	"time" // For Bot struct's CreatedAt field
)

// InternalUser represents a user interacting with the bot, potentially across different platforms.
type InternalUser struct {
	ID           int64
	FirstName    string // optional
	LastName     string // optional
	LanguageCode string
	SendMessages bool
	// BusinessConnectionID is optional, primarily used by business_bot.
	// For creator_bot, this will typically be empty.
	BusinessConnectionID string
}

// Bot represents a bot entity, managed by creator_bot and potentially referenced by business_bot.
// This struct was previously found in both creator_bot/repository/bot_repository.go
// and business_bot/repository/bot_repository.go.
type Bot struct {
	ID int64 `bson:"_id"` // Assuming BSON tags are still relevant if this struct is used with MongoDB

	Username    string `bson:"username"`
	SecretToken string `bson:"secret_token"`

	UserID    int64     `bson:"user_id"`    // The user who created/owns this bot
	CreatedAt time.Time `bson:"created_at"`
}

// Placeholder for other common types if any are identified later.
