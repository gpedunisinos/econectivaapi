package requests

type UpdateConnectionTypeInput struct {
	Name          string `json:"name" binding:"required,max=100"`
	StrokeStyle   string `json:"stroke_style" binding:"max=50"`
	IsDirectional *bool  `json:"is_directional"`
}
