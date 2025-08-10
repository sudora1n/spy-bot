package types

type MediaItemProcess struct {
	Type     string
	FileID   string
	FileSize int64
	Caption  string
}

type MediaItem struct {
	Type     string
	FileID   string
	FileSize int64
}

type MediaDiff struct {
	Added   *MediaItem
	Removed *MediaItem
}
