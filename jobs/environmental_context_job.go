package jobs

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/repository"
	"github.com/OdairPianta/unisinos_usp_api/services"
)

const EnvironmentalContextJob = "EnvironmentalContextJob"

type EnvironmentalContextJobPayload struct {
	LivingClueID uint `json:"living_clue_id"`
}

func RegisterEnvironmentalContextJob(livingClue *models.LivingClue) error {
	if livingClue.Latitude == nil || livingClue.Longitude == nil {
		return nil
	}
	if context, err := repository.ResetEnvironmentalContext(livingClue); err != nil {
		return err
	} else if context == nil {
		return nil
	}
	return RegisterJob(EnvironmentalContextJob, EnvironmentalContextJobPayload{LivingClueID: livingClue.ID})
}

func RunEnvironmentalContextJob(payload string) error {
	var data EnvironmentalContextJobPayload
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return err
	}
	livingClue, err := repository.FindLivingClueByID(fmt.Sprint(data.LivingClueID))
	if err != nil {
		return err
	}
	if livingClue.Latitude == nil || livingClue.Longitude == nil {
		return nil
	}
	context, err := repository.FindEnvironmentalContextByLivingClueID(data.LivingClueID)
	if err != nil {
		return err
	}
	if livingClue.Latitude == nil || livingClue.Longitude == nil {
		return nil
	}
	context.Status = models.EnvironmentalContextStatusProcessing
	context.ErrorMessage = ""
	if err := repository.SaveEnvironmentalContext(context); err != nil {
		return err
	}

	result, fetchErr := services.FetchEnvironmentalContext(*livingClue.Latitude, *livingClue.Longitude)
	if fetchErr != nil {
		log.Printf("environmental context enrichment failed for living_clue_id=%d: %v", livingClue.ID, fetchErr)
		context.Status = models.EnvironmentalContextStatusFailed
		context.ErrorMessage = fetchErr.Error()
		return repository.SaveEnvironmentalContext(context)
	}

	now := time.Now()
	context.Status = models.EnvironmentalContextStatusCompleted
	context.AnalyzedLatitude = *livingClue.Latitude
	context.AnalyzedLongitude = *livingClue.Longitude
	if result.Biome != "" {
		context.Biome = result.Biome
	}
	context.LandCover = result.LandCover
	context.VegetationPercent = result.VegetationPercent
	context.WaterDistanceM = result.WaterDistanceM
	context.AltitudeM = result.AltitudeM
	context.SlopePercent = result.SlopePercent
	context.BuiltUpPercent = result.BuiltUpPercent
	context.PopulationDensity = result.PopulationDensity
	context.Sources = result.Sources
	context.AnalyzedAt = &now
	context.ErrorMessage = ""
	return repository.SaveEnvironmentalContext(context)
}
