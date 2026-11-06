package jobs

import (
	"encoding/json"

	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/repository"
)

func RegisterJob(jobType string, payload interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	job := models.Job{
		Type:    jobType,
		Payload: string(payloadJSON),
		Status:  models.JobStatusPending,
	}

	err = repository.CreateJob(&job)
	if err != nil {
		return err
	}

	return nil
}
