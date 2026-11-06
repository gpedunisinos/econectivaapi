package requests

type UpdateNodeTypeInput struct {
	Name  string `json:"name" binding:"required,max=100"`
	Color string `json:"color" binding:"required,max=20"`
}
