package types

type InternalUser struct {
	ID                   int64
	FirstName            string // optional
	LastName             string // optional
	LanguageCode         string
	SendMessages         bool
	BusinessConnectionID string // optional
}

type SettingMeta struct {
	MessageID string
	Status    bool
	Data      int
}
