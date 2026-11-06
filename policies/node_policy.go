package policies

import "github.com/OdairPianta/unisinos_usp_api/models"

type NodePolicy struct {
	User *models.User
}

func NewNodePolicy(user *models.User) *NodePolicy {
	return &NodePolicy{User: user}
}

func (p *NodePolicy) ViewAny() bool {
	if p.User == nil {
		return false
	}
	return true
}

func (p *NodePolicy) View(model *models.Node) bool {
	if p.User == nil || model == nil {
		return false
	}
	return true
}

func (p *NodePolicy) Create() bool {
	if p.User == nil {
		return false
	}
	return true
}

func (p *NodePolicy) Delete(model *models.Node) bool {
	if p.User == nil || model == nil {
		return false
	}
	return true
}

func (p *NodePolicy) Update(model *models.Node) bool {
	if p.User == nil || model == nil {
		return false
	}
	return true
}
