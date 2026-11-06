package requests

type UpdateLivingClueInput struct {
	AliasNames               *[]string `json:"alias_names"`
	DescriptionFormatVersion uint      `json:"description_format_version"`
	// Omitted coordinates preserve the location; remove_location explicitly clears both.
	RemoveLocation   bool                   `json:"remove_location"`
	Name             string                 `json:"name" binding:"required,max=255"`
	CategoryID       uint                   `json:"category_id" binding:"required,exists=categories.id"`
	Description      string                 `json:"description"`
	DescriptionDelta string                 `json:"description_delta" binding:"omitempty,max=1000000"`
	Latitude         *float64               `json:"latitude" binding:"omitempty,latitude"`
	Longitude        *float64               `json:"longitude" binding:"omitempty,longitude"`
	ImageURL         string                 `json:"image_url" binding:"max=255"`
	Metadata         map[string]interface{} `json:"metadata"`
	IsActive         *bool                  `json:"is_active"`
}
