package policies

import "github.com/OdairPianta/unisinos_usp_api/models"

type NodeTypePolicy struct {
	User *models.User
}

func NewNodeTypePolicy(user *models.User) *NodeTypePolicy {
	return &NodeTypePolicy{User: user}
}

func (p *NodeTypePolicy) ViewAny() bool {
	if p.User == nil {
		return false
	}
	return true
}

func (p *NodeTypePolicy) View(model *models.NodeType) bool {
	if p.User == nil || model == nil {
		return false
	}
	return true
}

func (p *NodeTypePolicy) Create() bool {
	if p.User == nil {
		return false
	}
	return true
}

func (p *NodeTypePolicy) Delete(model *models.NodeType) bool {
	if p.User == nil || model == nil {
		return false
	}
	return true
}

func (p *NodeTypePolicy) Update(model *models.NodeType) bool {
	if p.User == nil || model == nil {
		return false
	}
	return true
}
