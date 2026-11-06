package models

type NodeLink struct {
	DefaultModel
	SourceNodeID uint  `gorm:"notNull;uniqueIndex:idx_node_link_pair" json:"source_node_id"`
	TargetNodeID uint  `gorm:"notNull;uniqueIndex:idx_node_link_pair" json:"target_node_id"`
	SourceNode   *Node `gorm:"foreignKey:SourceNodeID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"source_node"`
	TargetNode   *Node `gorm:"foreignKey:TargetNodeID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"target_node"`
}
