package requests

type CreateClueConnectionInput struct {
	SourceClueID     uint   `json:"source_clue_id" binding:"required,exists=living_clues.id"`
	TargetClueID     uint   `json:"target_clue_id" binding:"required,exists=living_clues.id"`
	ConnectionTypeID uint   `json:"connection_type_id" binding:"required,exists=connection_types.id"`
	Strength         *int   `json:"strength"`
	Evidence         string `json:"evidence"`
}
