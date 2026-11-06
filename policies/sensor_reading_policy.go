package policies

import "github.com/OdairPianta/unisinos_usp_api/models"

type SensorReadingPolicy struct {
	User *models.User
}

func NewSensorReadingPolicy(user *models.User) *SensorReadingPolicy {
	return &SensorReadingPolicy{User: user}
}

func (p *SensorReadingPolicy) ViewAny() bool {
	return p.User != nil
}

func (p *SensorReadingPolicy) View(model *models.SensorReading) bool {
	return p.User != nil && model != nil
}

func (p *SensorReadingPolicy) Delete(model *models.SensorReading) bool {
	return p.User != nil && model != nil
}
