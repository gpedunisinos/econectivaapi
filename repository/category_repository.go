package repository

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
)

func FindCategories(search string, limit *int, offset *int) ([]models.Category, error) {
	var categories []models.Category
	query := config.DB.Order("name")
	if search != "" {
		query = query.Where("name LIKE ? OR id LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if limit != nil && *limit > 0 {
		query = query.Limit(*limit)
	}
	if offset != nil && *offset > 0 {
		query = query.Offset(*offset)
	}
	err := query.Find(&categories).Error
	if err != nil {
		return nil, err
	}
	return categories, nil
}

func FindCategoryByID(ID string) (*models.Category, error) {
	var category models.Category
	err := config.DB.Where("id = ?", ID).First(&category).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

func FindCategoryByName(name string) (*models.Category, error) {
	var category models.Category
	err := config.DB.Where("name = ?", name).First(&category).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

func CreateCategory(category *models.Category) error {
	return config.DB.Create(&category).Error
}

func UpdateCategory(category *models.Category, updatedCategory *models.Category) error {
	return config.DB.Model(&category).Updates(updatedCategory).Error
}

func DeleteCategory(category *models.Category) error {
	return config.DB.Delete(&category).Error
}

func FindCategoryItemSelects(search string, limit *int, offset *int) ([]models.ItemSelect, error) {
	categories, err := FindCategories(search, limit, offset)
	if err != nil {
		return nil, err
	}
	var itemSelects []models.ItemSelect
	for _, category := range categories {
		itemSelects = append(itemSelects, models.ItemSelect{
			ID:   category.ID,
			Name: category.Name,
		})
	}
	return itemSelects, nil
}
