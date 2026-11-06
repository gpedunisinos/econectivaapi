package controllers

import (
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/OdairPianta/unisinos_usp_api/http/requests"
	"github.com/OdairPianta/unisinos_usp_api/jobs"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/policies"
	"github.com/OdairPianta/unisinos_usp_api/repository"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

func livingClueQueryUint(context *gin.Context, key string) *uint {
	if raw := context.Query(key); raw != "" {
		if parsed, err := strconv.ParseUint(raw, 10, 64); err == nil {
			value := uint(parsed)
			return &value
		}
	}
	return nil
}

// @Summary Get living clues
// @Description Get living clues with optional search, pagination and geospatial filters
// @Tags living_clues
// @Accept json
// @Produce json
// @Param search query string false "Search"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param category_id query int false "Category ID"
// @Param north query number false "North boundary"
// @Param south query number false "South boundary"
// @Param east query number false "East boundary"
// @Param west query number false "West boundary"
// @Param lat query number false "Latitude"
// @Param lng query number false "Longitude"
// @Param radius query number false "Radius in meters"
// @Success 200 {array} models.LivingClue
// @Failure 400 {object} models.APIMessage "Bad request"
// @Router /living_clues [get]
// @Security Bearer
func FindLivingClues(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	if !policies.NewLivingCluePolicy(&user).ViewAny() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	search := context.Query("search")
	limit, _ := strconv.Atoi(context.Query("limit"))
	offset, _ := strconv.Atoi(context.Query("offset"))
	categoryID := livingClueQueryUint(context, "category_id")

	northParam := context.Query("north")
	southParam := context.Query("south")
	eastParam := context.Query("east")
	westParam := context.Query("west")
	latParam := context.Query("lat")
	lngParam := context.Query("lng")
	radiusParam := context.Query("radius")

	if latParam != "" || lngParam != "" || radiusParam != "" {
		if latParam == "" || lngParam == "" || radiusParam == "" {
			context.JSON(http.StatusBadRequest, gin.H{"message": "Os parâmetros lat, lng e radius são obrigatórios."})
			return
		}
		latitude, errLat := strconv.ParseFloat(latParam, 64)
		longitude, errLng := strconv.ParseFloat(lngParam, 64)
		radius, errRadius := strconv.ParseFloat(radiusParam, 64)
		if errLat != nil || errLng != nil || errRadius != nil || radius <= 0 {
			context.JSON(http.StatusBadRequest, gin.H{"message": "Os parâmetros lat, lng e radius devem ser números válidos e radius deve ser maior que zero."})
			return
		}
		livingClues, err := repository.FindLivingCluesByRadius(search, categoryID, latitude, longitude, radius, &limit, &offset)
		if err != nil {
			sentry.CaptureException(err)
			context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
			return
		}
		context.JSON(http.StatusOK, livingClues)
		return
	}

	if northParam != "" || southParam != "" || eastParam != "" || westParam != "" {
		if northParam == "" || southParam == "" || eastParam == "" || westParam == "" {
			context.JSON(http.StatusBadRequest, gin.H{"message": "Os parâmetros north, south, east e west são obrigatórios."})
			return
		}
		north, errNorth := strconv.ParseFloat(northParam, 64)
		south, errSouth := strconv.ParseFloat(southParam, 64)
		east, errEast := strconv.ParseFloat(eastParam, 64)
		west, errWest := strconv.ParseFloat(westParam, 64)
		if errNorth != nil || errSouth != nil || errEast != nil || errWest != nil {
			context.JSON(http.StatusBadRequest, gin.H{"message": "Os parâmetros north, south, east e west devem ser números válidos."})
			return
		}
		if north < south {
			context.JSON(http.StatusBadRequest, gin.H{"message": "O parâmetro north deve ser maior ou igual ao south."})
			return
		}
		if east < west {
			context.JSON(http.StatusBadRequest, gin.H{"message": "O parâmetro east deve ser maior ou igual ao west."})
			return
		}
		livingClues, err := repository.FindLivingCluesByBounds(search, categoryID, north, south, east, west, &limit, &offset)
		if err != nil {
			sentry.CaptureException(err)
			context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
			return
		}
		context.JSON(http.StatusOK, livingClues)
		return
	}

	livingClues, err := repository.FindLivingClues(search, categoryID, &limit, &offset)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}
	context.JSON(http.StatusOK, livingClues)
}

// @Summary Get living clue
// @Description Get a living clue by id including the last 10 sensor readings
// @Tags living_clues
// @Accept json
// @Produce json
// @Param id path int true "Living Clue ID"
// @Success 200 {object} models.LivingClue
// @Failure 400 {object} models.APIMessage "Bad request"
// @Router /living_clues/{id} [get]
// @Security Bearer
func FindLivingClue(context *gin.Context) {
	livingClue, err := repository.FindLivingClueByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Pista viva não encontrada!"})
		return
	}
	user := context.MustGet("user").(models.User)
	if !policies.NewLivingCluePolicy(&user).View(livingClue) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}
	context.JSON(http.StatusOK, livingClue)
}

// @Summary Add a living clue
// @Description Create a living clue with an optional coordinate pair (null/null or valid numbers, including zero). Supports text and explicit YouTube video Delta blocks.
// @Tags living_clues
// @Accept json
// @Produce json
// @Param living_clue body requests.CreateLivingClueInput true "Living Clue"
// @Success 200 {object} models.LivingClue
// @Failure 400 {object} models.APIMessage "Bad request"
// @Router /living_clues [post]
// @Security Bearer
func CreateLivingClue(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	if !policies.NewLivingCluePolicy(&user).Create() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}
	var input requests.CreateLivingClueInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}
	if err := services.ValidateQuillDelta(input.DescriptionDelta); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	if !validClueCoordinates(input.Latitude, input.Longitude) {
		context.JSON(400, gin.H{"message": "Informe ambas as coordenadas válidas ou deixe ambas vazias."})
		return
	}
	if input.DescriptionDelta != "" {
		input.Description = services.QuillPlainText(input.DescriptionDelta)
	}
	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}
	livingClue := models.LivingClue{Name: input.Name, CategoryID: input.CategoryID, Description: input.Description, DescriptionDelta: input.DescriptionDelta, Latitude: input.Latitude, Longitude: input.Longitude, ImageURL: input.ImageURL, Metadata: models.JSONMap(input.Metadata), IsActive: isActive}
	if err := repository.CreateLivingClue(&livingClue); err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}
	createdLivingClue, err := repository.FindLivingClueByID(fmt.Sprint(livingClue.ID))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusOK, livingClue)
		return
	}
	if err := jobs.RegisterEnvironmentalContextJob(createdLivingClue); err != nil {
		sentry.CaptureException(err)
	} else {
		if refreshed, refreshErr := repository.FindLivingClueByID(fmt.Sprint(livingClue.ID)); refreshErr == nil {
			createdLivingClue = refreshed
		}
	}
	context.JSON(http.StatusOK, createdLivingClue)
}

// @Summary Update a living clue
// @Description Update a living clue. Coordinates are an optional pair; omission preserves location, remove_location=true clears it. Zero is valid. description is derived from a supplied description_delta.
// @Tags living_clues
// @Accept json
// @Produce json
// @Param id path int true "Living Clue ID"
// @Param living_clue body requests.UpdateLivingClueInput true "Living Clue"
// @Success 200 {object} models.LivingClue
// @Failure 400 {object} models.APIMessage "Bad request"
// @Router /living_clues/{id} [put]
// @Security Bearer
func UpdateLivingClue(context *gin.Context) {
	livingClue, err := repository.FindLivingClueByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Pista viva não encontrada!"})
		return
	}
	user := context.MustGet("user").(models.User)
	if !policies.NewLivingCluePolicy(&user).Update(livingClue) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}
	var input requests.UpdateLivingClueInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}
	if services.HasExtendedQuillBlocks(livingClue.DescriptionDelta) && input.DescriptionFormatVersion < 1 {
		context.JSON(http.StatusConflict, gin.H{"message": "Esta pista contém integrações. Atualize a plataforma antes de editar para preservar o documento."})
		return
	}
	if err := services.ValidateQuillDelta(input.DescriptionDelta); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	if input.RemoveLocation {
		input.Latitude = nil
		input.Longitude = nil
	} else if input.Latitude == nil && input.Longitude == nil {
		input.Latitude = livingClue.Latitude
		input.Longitude = livingClue.Longitude
	}
	if !validClueCoordinates(input.Latitude, input.Longitude) {
		context.JSON(400, gin.H{"message": "Informe ambas as coordenadas válidas ou remova a localização."})
		return
	}
	if input.DescriptionDelta != "" {
		input.Description = services.QuillPlainText(input.DescriptionDelta)
	}
	isActive := livingClue.IsActive
	if input.IsActive != nil {
		isActive = *input.IsActive
	}
	locationChanged := !sameCoordinate(livingClue.Latitude, input.Latitude) || !sameCoordinate(livingClue.Longitude, input.Longitude)
	updatedLivingClue := models.LivingClue{Name: input.Name, CategoryID: input.CategoryID, Description: input.Description, DescriptionDelta: input.DescriptionDelta, Latitude: input.Latitude, Longitude: input.Longitude, ImageURL: input.ImageURL, Metadata: models.JSONMap(input.Metadata), IsActive: isActive}
	if err = repository.UpdateLivingClueWithEdit(livingClue, &updatedLivingClue, user.ID, locationChanged); err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}
	refreshedLivingClue, err := repository.FindLivingClueByID(fmt.Sprint(livingClue.ID))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusOK, updatedLivingClue)
		return
	}
	if locationChanged {
		if err := jobs.RegisterEnvironmentalContextJob(refreshedLivingClue); err != nil {
			sentry.CaptureException(err)
		} else {
			if reloaded, reloadErr := repository.FindLivingClueByID(fmt.Sprint(livingClue.ID)); reloadErr == nil {
				refreshedLivingClue = reloaded
			}
		}
	}
	context.JSON(http.StatusOK, refreshedLivingClue)
}

// @Summary Reprocess living clue environmental context
// @Description Queue a new environmental analysis for a living clue
// @Tags living_clues
// @Produce json
// @Param id path int true "Living Clue ID"
// @Success 202 {object} models.EnvironmentalContext
// @Failure 404 {object} models.APIMessage "Not found"
// @Router /living_clues/{id}/environmental_context/reprocess [post]
// @Security Bearer
func ReprocessLivingClueEnvironmentalContext(context *gin.Context) {
	livingClue, err := repository.FindLivingClueByID(context.Param("id"))
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"message": "Pista viva não encontrada!"})
		return
	}
	user := context.MustGet("user").(models.User)
	if !policies.NewLivingCluePolicy(&user).Update(livingClue) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}
	if err := jobs.RegisterEnvironmentalContextJob(livingClue); err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}
	environmentalContext, _ := repository.FindEnvironmentalContextByLivingClueID(livingClue.ID)
	context.JSON(http.StatusAccepted, environmentalContext)
}

// @Summary Delete a living clue
// @Description Delete a living clue by id
// @Tags living_clues
// @Accept json
// @Produce json
// @Param id path int true "Living Clue ID"
// @Success 200 {object} models.LivingClue
// @Failure 400 {object} models.APIMessage "Bad request"
// @Router /living_clues/{id} [delete]
// @Security Bearer
func DeleteLivingClue(context *gin.Context) {
	livingClue, err := repository.FindLivingClueByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Pista viva não encontrada!"})
		return
	}
	user := context.MustGet("user").(models.User)
	if !policies.NewLivingCluePolicy(&user).Delete(livingClue) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}
	if err = repository.DeleteLivingClue(livingClue); err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}
	context.JSON(http.StatusOK, livingClue)
}

// @Summary Select living clues
// @Description Select living clues for dropdowns
// @Tags living_clues
// @Accept json
// @Produce json
// @Param search query string false "Search"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param category_id query int false "Category ID"
// @Success 200 {array} models.ItemSelect
// @Failure 400 {object} models.APIMessage "Bad request"
// @Router /living_clues_select [get]
// @Security Bearer
func SelectLivingClues(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	if !policies.NewLivingCluePolicy(&user).ViewAny() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}
	search := context.Query("search")
	limit, _ := strconv.Atoi(context.Query("limit"))
	offset, _ := strconv.Atoi(context.Query("offset"))
	itemSelects, err := repository.FindLivingClueSelects(search, livingClueQueryUint(context, "category_id"), &limit, &offset)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}
	context.JSON(http.StatusOK, itemSelects)
}

func sameCoordinate(a, b *float64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
func validClueCoordinates(lat, lng *float64) bool {
	if lat == nil || lng == nil {
		return lat == nil && lng == nil
	}
	return !math.IsNaN(*lat) && !math.IsNaN(*lng) && !math.IsInf(*lat, 0) && !math.IsInf(*lng, 0) && *lat >= -90 && *lat <= 90 && *lng >= -180 && *lng <= 180
}
