package repository

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
)

func FindSampleDetails(search string, limit *int, offset *int) ([]models.SampleDetail, error) {
	var sampleDetails []models.SampleDetail

	query := config.DB.Order("sample_string")
	if search != "" {
		query = query.Where("id LIKE ? OR sample_string LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if limit != nil && *limit > 0 {
		query = query.Limit(*limit)
	}
	if offset != nil && *offset > 0 {
		query = query.Offset(*offset)
	}

	if err := query.Find(&sampleDetails).Error; err != nil {
		return nil, err
	}

	return sampleDetails, nil
}

func FindSampleDetailSelects(search string, limit *int, offset *int) ([]models.ItemSelect, error) {
	sampleDetails, err := FindSampleDetails(search, limit, offset)
	if err != nil {
		return nil, err
	}

	var itemSelects []models.ItemSelect
	for _, detail := range sampleDetails {
		itemSelects = append(itemSelects, models.ItemSelect{
			ID:   detail.ID,
			Name: detail.SampleString,
		})
	}

	return itemSelects, nil
}
