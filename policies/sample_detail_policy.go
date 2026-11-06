package policies

import "github.com/OdairPianta/unisinos_usp_api/models"

type SampleDetailPolicy struct {
	User *models.User
}

func NewSampleDetailPolicy(user *models.User) *SampleDetailPolicy {
	return &SampleDetailPolicy{User: user}
}

func (p *SampleDetailPolicy) ViewAny() bool {
	return p.User != nil
}

func (p *SampleDetailPolicy) View(model *models.SampleDetail) bool {
	return p.User != nil && model != nil
}

func (p *SampleDetailPolicy) Create() bool {
	return p.User != nil
}

func (p *SampleDetailPolicy) Delete(model *models.SampleDetail) bool {
	return p.User != nil && model != nil
}

func (p *SampleDetailPolicy) Update(model *models.SampleDetail) bool {
	return p.User != nil && model != nil
}
