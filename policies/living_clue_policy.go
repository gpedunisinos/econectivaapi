package policies

import "github.com/OdairPianta/unisinos_usp_api/models"

type LivingCluePolicy struct {
	User *models.User
}

func NewLivingCluePolicy(user *models.User) *LivingCluePolicy {
	return &LivingCluePolicy{User: user}
}

func (p *LivingCluePolicy) ViewAny() bool {
	if p.User == nil {
		return false
	}
	return true
}

func (p *LivingCluePolicy) View(model *models.LivingClue) bool {
	if p.User == nil || model == nil {
		return false
	}
	return true
}

func (p *LivingCluePolicy) Create() bool {
	if p.User == nil {
		return false
	}
	return true
}

func (p *LivingCluePolicy) Delete(model *models.LivingClue) bool {
	if p.User == nil || model == nil {
		return false
	}
	return true
}

func (p *LivingCluePolicy) Update(model *models.LivingClue) bool {
	if p.User == nil || model == nil {
		return false
	}
	return true
}
