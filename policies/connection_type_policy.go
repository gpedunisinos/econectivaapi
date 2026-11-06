package policies

import "github.com/OdairPianta/unisinos_usp_api/models"

type ConnectionTypePolicy struct {
	User *models.User
}

func NewConnectionTypePolicy(user *models.User) *ConnectionTypePolicy {
	return &ConnectionTypePolicy{User: user}
}

func (p *ConnectionTypePolicy) ViewAny() bool {
	if p.User == nil {
		return false
	}
	return true
}

func (p *ConnectionTypePolicy) View(model *models.ConnectionType) bool {
	if p.User == nil || model == nil {
		return false
	}
	return true
}

func (p *ConnectionTypePolicy) Create() bool {
	if p.User == nil {
		return false
	}
	return true
}

func (p *ConnectionTypePolicy) Delete(model *models.ConnectionType) bool {
	if p.User == nil || model == nil {
		return false
	}
	return true
}

func (p *ConnectionTypePolicy) Update(model *models.ConnectionType) bool {
	if p.User == nil || model == nil {
		return false
	}
	return true
}
