package domain

type SourceType string

const (
	SourceTypeYouTube SourceType = "youtube"
)

func IsValidSourceType(typ string) bool {
	return typ == string(SourceTypeYouTube)
}

type Source struct {
	ID       string
	Type     SourceType
	Metadata map[string]interface{}
}
