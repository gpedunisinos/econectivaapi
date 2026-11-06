package controllers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/OdairPianta/unisinos_usp_api/http/requests"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/policies"
	"github.com/OdairPianta/unisinos_usp_api/repository"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func FindCategories(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	if !policies.NewCategoryPolicy(&user).ViewAny() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	search := context.Query("search")
	limit, _ := strconv.Atoi(context.Query("limit"))
	offset, _ := strconv.Atoi(context.Query("offset"))

	categories, err := repository.FindCategories(search, &limit, &offset)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, categories)
}

func FindCategory(context *gin.Context) {
	category, err := repository.FindCategoryByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Categoria não encontrada!"})
		return
	}

	user := context.MustGet("user").(models.User)
	if !policies.NewCategoryPolicy(&user).View(category) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	context.JSON(http.StatusOK, category)
}

func CreateCategory(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	if !policies.NewCategoryPolicy(&user).Create() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	var input requests.CreateCategoryInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}

	_, err := repository.FindCategoryByName(input.Name)
	if err == nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Categoria já cadastrada!"})
		return
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	category := models.Category{
		Name:         input.Name,
		Type:         input.Type,
		ColorHex:     input.ColorHex,
		IconCode:     input.IconCode,
		Description:  input.Description,
		MetadataTags: input.MetadataTags,
	}
	err = repository.CreateCategory(&category)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, category)
}

func UpdateCategory(context *gin.Context) {
	category, err := repository.FindCategoryByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Categoria não encontrada!"})
		return
	}

	user := context.MustGet("user").(models.User)
	if !policies.NewCategoryPolicy(&user).Update(category) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	var input requests.UpdateCategoryInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}

	existsCategory, err := repository.FindCategoryByName(input.Name)
	if err == nil && existsCategory.ID != category.ID {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Categoria já cadastrada!"})
		return
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	updatedCategory := models.Category{
		Name:         input.Name,
		Type:         input.Type,
		ColorHex:     input.ColorHex,
		IconCode:     input.IconCode,
		Description:  input.Description,
		MetadataTags: input.MetadataTags,
	}

	err = repository.UpdateCategory(category, &updatedCategory)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	category.Name = updatedCategory.Name
	category.Type = updatedCategory.Type
	category.ColorHex = updatedCategory.ColorHex
	category.IconCode = updatedCategory.IconCode
	category.Description = updatedCategory.Description
	category.MetadataTags = updatedCategory.MetadataTags

	context.JSON(http.StatusOK, category)
}

func DeleteCategory(context *gin.Context) {
	category, err := repository.FindCategoryByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Categoria não encontrada!"})
		return
	}

	user := context.MustGet("user").(models.User)
	if !policies.NewCategoryPolicy(&user).Delete(category) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	err = repository.DeleteCategory(category)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, category)
}

func SelectCategories(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	if !policies.NewCategoryPolicy(&user).ViewAny() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	search := context.Query("search")
	limit, _ := strconv.Atoi(context.Query("limit"))
	offset, _ := strconv.Atoi(context.Query("offset"))

	itemSelects, err := repository.FindCategoryItemSelects(search, &limit, &offset)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, itemSelects)
}
