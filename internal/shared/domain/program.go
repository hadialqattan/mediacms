package domain

import "time"

type ProgramType string

const (
	ProgramTypePodcast     ProgramType = "podcast"
	ProgramTypeDocumentary ProgramType = "documentary"
)

func IsValidProgramType(typ string) bool {
	switch typ {
	case string(ProgramTypePodcast), string(ProgramTypeDocumentary):
		return true
	default:
		return false
	}
}

type ProgramLanguage string

const (
	LanguageAr ProgramLanguage = "ar"
	LanguageEn ProgramLanguage = "en"
)

func IsValidProgramLanguage(lang string) bool {
	switch lang {
	case string(LanguageAr), string(LanguageEn):
		return true
	default:
		return false
	}
}

type Program struct {
	ID          string          `json:"id"`
	Slug        string          `json:"slug"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Type        ProgramType     `json:"type"`
	Language    ProgramLanguage `json:"language"`
	DurationMs  int             `json:"duration_ms"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   *time.Time      `json:"updated_at,omitempty"`
	PublishedAt *time.Time      `json:"published_at,omitempty"`
	DeletedAt   *time.Time      `json:"deleted_at,omitempty"`
	SourceID    *string         `json:"source_id,omitempty"`
	CreatedBy   string          `json:"created_by"`
	PublishedBy *string         `json:"published_by,omitempty"`
	UpdatedBy   *string         `json:"updated_by,omitempty"`
	DeletedBy   *string         `json:"deleted_by,omitempty"`
	Categories  []Category      `json:"categories"`
}

func (p *Program) IsPublished() bool {
	return p.PublishedAt != nil && p.DeletedAt == nil
}

func (p *Program) IsDeleted() bool {
	return p.DeletedAt != nil
}
