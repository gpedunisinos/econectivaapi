package controllers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/policies"
	"github.com/OdairPianta/unisinos_usp_api/repository"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

// @Summary Get sensor readings
// @Description Get all sensor readings
// @Tags sensor_readings
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {array} models.SensorReading
// @Failure 400 {object} models.APIMessage "Bad request"
// @Router /sensor_readings [get]
// @Security Bearer
func FindSensorReadings(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	if !policies.NewSensorReadingPolicy(&user).ViewAny() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	limit, _ := strconv.Atoi(context.Query("limit"))
	offset, _ := strconv.Atoi(context.Query("offset"))

	sensorReadings, err := repository.FindSensorReadings(&limit, &offset)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, sensorReadings)
}

// @Summary Get sensor reading
// @Description Get a sensor reading by id
// @Tags sensor_readings
// @Accept json
// @Produce json
// @Param id path int true "Sensor Reading ID"
// @Success 200 {object} models.SensorReading
// @Failure 400 {object} models.APIMessage "Bad request"
// @Router /sensor_readings/{id} [get]
// @Security Bearer
func FindSensorReading(context *gin.Context) {
	sensorReading, err := repository.FindSensorReadingByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Leitura de sensor não encontrada!"})
		return
	}

	user := context.MustGet("user").(models.User)
	if !policies.NewSensorReadingPolicy(&user).View(sensorReading) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	context.JSON(http.StatusOK, sensorReading)
}

// @Summary Create sensor reading
// @Description Create a public sensor reading linked to a living clue using any payload format
// @Tags sensor_readings
// @Accept json,plain,mpfd,x-www-form-urlencoded
// @Produce json
// @Param id path int true "Living Clue ID"
// @Param payload body string false "Raw payload"
// @Success 201 {object} models.SensorReading
// @Failure 400 {object} models.APIMessage "Bad request"
// @Router /living_clues/{id}/sensor_readings [post]
func CreateSensorReading(context *gin.Context) {
	livingClue, err := repository.FindLivingClueByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Pista viva não encontrada!"})
		return
	}

	body, err := context.GetRawData()
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusBadRequest, gin.H{"message": "Não foi possível ler o corpo da requisição."})
		return
	}

	sensorReading := models.SensorReading{
		LivingClueID: livingClue.ID,
		ContentType:  context.GetHeader("Content-Type"),
		PayloadRaw:   string(body),
	}

	if err = repository.CreateSensorReading(&sensorReading); err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusCreated, sensorReading)
}

// @Summary Delete sensor reading
// @Description Delete a sensor reading by id
// @Tags sensor_readings
// @Accept json
// @Produce json
// @Param id path int true "Sensor Reading ID"
// @Success 200 {object} models.SensorReading
// @Failure 400 {object} models.APIMessage "Bad request"
// @Router /sensor_readings/{id} [delete]
// @Security Bearer
func DeleteSensorReading(context *gin.Context) {
	sensorReading, err := repository.FindSensorReadingByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Leitura de sensor não encontrada!"})
		return
	}

	user := context.MustGet("user").(models.User)
	if !policies.NewSensorReadingPolicy(&user).Delete(sensorReading) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	if err = repository.DeleteSensorReading(sensorReading); err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, sensorReading)
}
