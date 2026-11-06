package policies

import "github.com/OdairPianta/unisinos_usp_api/models"

type NodeLinkPolicy struct {
	User *models.User
}

func NewNodeLinkPolicy(user *models.User) *NodeLinkPolicy {
	return &NodeLinkPolicy{User: user}
}

func (p *NodeLinkPolicy) ViewAny() bool {
	if p.User == nil {
		return false
	}
	return true
}

func (p *NodeLinkPolicy) View(model *models.NodeLink) bool {
	if p.User == nil || model == nil {
		return false
	}
	return true
}

func (p *NodeLinkPolicy) Create() bool {
	if p.User == nil {
		return false
	}
	return true
}

func (p *NodeLinkPolicy) Delete(model *models.NodeLink) bool {
	if p.User == nil || model == nil {
		return false
	}
	return true
}

func (p *NodeLinkPolicy) Update(model *models.NodeLink) bool {
	if p.User == nil || model == nil {
		return false
	}
	return true
}
