package repository

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
)

func CreateSensorReading(sensorReading *models.SensorReading) error {
	return config.DB.Create(&sensorReading).Error
}

func FindSensorReadings(limit *int, offset *int) ([]models.SensorReading, error) {
	var sensorReadings []models.SensorReading
	query := config.DB.Order("id desc")
	if limit != nil && *limit > 0 {
		query = query.Limit(*limit)
	}
	if offset != nil && *offset > 0 {
		query = query.Offset(*offset)
	}
	if err := query.Find(&sensorReadings).Error; err != nil {
		return nil, err
	}
	return sensorReadings, nil
}

func FindSensorReadingByID(id string) (*models.SensorReading, error) {
	var sensorReading models.SensorReading
	if err := config.DB.Where("sensor_readings.id = ?", id).First(&sensorReading).Error; err != nil {
		return nil, err
	}
	return &sensorReading, nil
}

func FindSensorReadingsByLivingClueID(livingClueID uint) ([]models.SensorReading, error) {
	var sensorReadings []models.SensorReading
	if err := config.DB.Where("living_clue_id = ?", livingClueID).Order("id desc").Find(&sensorReadings).Error; err != nil {
		return nil, err
	}
	return sensorReadings, nil
}

func FindLastSensorReadingsByLivingClueID(livingClueID uint, limit int) ([]models.SensorReading, error) {
	var sensorReadings []models.SensorReading
	query := config.DB.Where("living_clue_id = ?", livingClueID).Order("id desc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&sensorReadings).Error; err != nil {
		return nil, err
	}
	return sensorReadings, nil
}

func DeleteSensorReading(sensorReading *models.SensorReading) error {
	return config.DB.Delete(&sensorReading).Error
}
