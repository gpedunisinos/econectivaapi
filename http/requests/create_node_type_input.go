package requests

type CreateNodeTypeInput struct {
	Name  string `json:"name" binding:"required,not_exists=node_types.name,max=100"`
	Color string `json:"color" binding:"required,max=20"`
}
