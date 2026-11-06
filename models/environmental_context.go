package models

import "time"

const (
	EnvironmentalContextStatusPending    = "pending"
	EnvironmentalContextStatusProcessing = "processing"
	EnvironmentalContextStatusCompleted  = "completed"
	EnvironmentalContextStatusFailed     = "failed"
)

// EnvironmentalContext stores the versioned territorial snapshot calculated
// from a living clue coordinate.
type EnvironmentalContext struct {
	DefaultModel
	LivingClueID      uint       `gorm:"notNull;uniqueIndex" json:"living_clue_id"`
	Status            string     `gorm:"size:20;notNull;default:pending" json:"status"`
	AnalyzedLatitude  float64    `gorm:"notNull" json:"analyzed_latitude"`
	AnalyzedLongitude float64    `gorm:"notNull" json:"analyzed_longitude"`
	Biome             string     `gorm:"size:100" json:"biome"`
	LandCover         string     `gorm:"size:150" json:"land_cover"`
	VegetationPercent *float64   `json:"vegetation_percent"`
	WaterDistanceM    *float64   `json:"water_distance_m"`
	AltitudeM         *float64   `json:"altitude_m"`
	SlopePercent      *float64   `json:"slope_percent"`
	BuiltUpPercent    *float64   `json:"built_up_percent"`
	PopulationDensity *float64   `json:"population_density_km2"`
	Sources           JSONMap    `gorm:"type:json" json:"sources"`
	AnalyzedAt        *time.Time `json:"analyzed_at"`
	ErrorMessage      string     `gorm:"type:text" json:"error_message,omitempty"`
}
