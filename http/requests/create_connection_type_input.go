package requests

type CreateConnectionTypeInput struct {
	Name          string `json:"name" binding:"required,not_exists=connection_types.name,max=100"`
	StrokeStyle   string `json:"stroke_style" binding:"max=50"`
	IsDirectional *bool  `json:"is_directional"`
}
