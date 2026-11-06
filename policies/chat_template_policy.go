package policies

import (
	"github.com/OdairPianta/unisinos_usp_api/enums"
	"github.com/OdairPianta/unisinos_usp_api/models"
)

type ChatTemplatePolicy struct {
	User *models.User
}

func NewChatTemplatePolicy(user *models.User) *ChatTemplatePolicy {
	return &ChatTemplatePolicy{User: user}
}

func (p *ChatTemplatePolicy) ViewAny() bool {
	return p.User != nil
}

func (p *ChatTemplatePolicy) View(model *models.ChatTemplate) bool {
	return p.User != nil && model != nil
}

func (p *ChatTemplatePolicy) Create() bool {
	return p.User != nil
}

func (p *ChatTemplatePolicy) Delete(model *models.ChatTemplate) bool {
	if p.User == nil || model == nil {
		return false
	}
	return p.User.ID == model.UserID || p.User.Profile == enums.UserProfileEnumAdministrator
}

func (p *ChatTemplatePolicy) Update(model *models.ChatTemplate) bool {
	if p.User == nil || model == nil {
		return false
	}
	return p.User.ID == model.UserID || p.User.Profile == enums.UserProfileEnumAdministrator
}
