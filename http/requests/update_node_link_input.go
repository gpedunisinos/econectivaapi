package requests

type UpdateNodeLinkInput struct {
	SourceNodeID uint `json:"source_node_id" binding:"required,exists=nodes.id"`
	TargetNodeID uint `json:"target_node_id" binding:"required,exists=nodes.id"`
}
