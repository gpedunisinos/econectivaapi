package policies

import "github.com/OdairPianta/unisinos_usp_api/models"

type ClueConnectionPolicy struct {
	User *models.User
}

func NewClueConnectionPolicy(user *models.User) *ClueConnectionPolicy {
	return &ClueConnectionPolicy{User: user}
}

func (p *ClueConnectionPolicy) ViewAny() bool {
	if p.User == nil {
		return false
	}
	return true
}

func (p *ClueConnectionPolicy) View(model *models.ClueConnection) bool {
	if p.User == nil || model == nil {
		return false
	}
	return true
}

func (p *ClueConnectionPolicy) Create() bool {
	if p.User == nil {
		return false
	}
	return true
}

func (p *ClueConnectionPolicy) Delete(model *models.ClueConnection) bool {
	if p.User == nil || model == nil {
		return false
	}
	return true
}

func (p *ClueConnectionPolicy) Update(model *models.ClueConnection) bool {
	if p.User == nil || model == nil {
		return false
	}
	return true
}
