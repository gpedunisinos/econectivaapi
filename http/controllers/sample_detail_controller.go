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

// SelectSamples returns sample details formatted for select inputs.
// @Summary Select sample details
// @Description Retrieve sample details for select inputs
// @Tags sample_details
// @Accept json
// @Produce json
// @Param search query string false "Search by id or sample_string"
// @Param limit query int false "Limit the number of results"
// @Param offset query int false "Offset the results"
// @Success 200 {array} models.ItemSelect "ok"
// @Failure 400 {object} models.APIMessage "Bad request"
// @Failure 404 {object} models.APIMessage "Not found"
// @Router /sample_details_select [get]
// @Security Bearer
func SelectSamples(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	if !policies.NewSampleDetailPolicy(&user).ViewAny() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	search := context.Query("search")
	limit, _ := strconv.Atoi(context.Query("limit"))
	offset, _ := strconv.Atoi(context.Query("offset"))

	itemSelects, err := repository.FindSampleDetailSelects(search, &limit, &offset)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, itemSelects)
}
