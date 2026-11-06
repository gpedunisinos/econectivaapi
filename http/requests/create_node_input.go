package requests

type CreateNodeInput struct {
	Name       string   `json:"name" binding:"required,max=255"`
	Color      string   `json:"color" binding:"required,max=20"`
	PositionX  *float64 `json:"x"`
	PositionY  *float64 `json:"y"`
	NodeTypeID uint     `json:"node_type_id" binding:"required,exists=node_types.id"`
}
