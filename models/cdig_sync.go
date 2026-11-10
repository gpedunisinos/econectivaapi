package models

import "time"

// Source includes both the configured environment key and origin URL. An ID from
// staging can never silently update a production source mapping.
type CDIGMapping struct {
	ID          uint   `gorm:"primaryKey"`
	Source      string `gorm:"size:64;uniqueIndex:idx_cdig_identity"`
	Kind        string `gorm:"size:24;uniqueIndex:idx_cdig_identity"`
	ExternalID  string `gorm:"size:64;uniqueIndex:idx_cdig_identity"`
	ClueID      uint   `gorm:"index"`
	RemoteHash  string `gorm:"size:64"`
	AppliedHash string `gorm:"size:64"`
	Suppressed  bool
	Missing     bool
	LastSeenAt  time.Time
}

// Only connections created by the importer are governed by it. Local links and
// the independent member clues are never cascade-deleted on a series update.
type CDIGMembership struct {
	ID           uint   `gorm:"primaryKey"`
	Source       string `gorm:"size:64;uniqueIndex:idx_cdig_membership"`
	SerieID      string `gorm:"size:64;uniqueIndex:idx_cdig_membership"`
	VideoID      string `gorm:"size:64;uniqueIndex:idx_cdig_membership"`
	ConnectionID uint
}
