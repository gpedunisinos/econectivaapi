package policies

import "github.com/OdairPianta/unisinos_usp_api/models"

type CategoryPolicy struct {
	User *models.User
}

func NewCategoryPolicy(user *models.User) *CategoryPolicy {
	return &CategoryPolicy{User: user}
}

func (p *CategoryPolicy) ViewAny() bool {
	if p.User == nil {
		return false
	}
	return true
}

func (p *CategoryPolicy) View(model *models.Category) bool {
	if p.User == nil || model == nil {
		return false
	}
	return true
}

func (p *CategoryPolicy) Create() bool {
	if p.User == nil {
		return false
	}
	return true
}

func (p *CategoryPolicy) Delete(model *models.Category) bool {
	if p.User == nil || model == nil {
		return false
	}
	return true
}

func (p *CategoryPolicy) Update(model *models.Category) bool {
	if p.User == nil || model == nil {
		return false
	}
	return true
}
