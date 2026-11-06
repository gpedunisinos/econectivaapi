package repository

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
)

func FindPendingJobs() (*[]models.Job, error) {
	var jobs []models.Job
	err := config.DB.Where("status = ?", models.JobStatusPending).Find(&jobs).Error
	if err != nil {
		return nil, err
	}
	return &jobs, nil
}

func CreateJob(job *models.Job) error {
	return config.DB.Create(&job).Error
}

func DeleteJob(job *models.Job) error {
	return config.DB.Delete(job).Error
}

func CreateFailedJob(failedJob *models.FailedJob) error {
	return config.DB.Create(&failedJob).Error
}
