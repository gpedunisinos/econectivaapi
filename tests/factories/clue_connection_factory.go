package factories

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
)

func CreateClueConnection() (*models.ClueConnection, error) {
	sourceClue, err := CreateLivingClue()
	if err != nil {
		return nil, err
	}

	targetClue, err := CreateLivingClue()
	if err != nil {
		return nil, err
	}

	connectionType, err := CreateConnectionType()
	if err != nil {
		return nil, err
	}

	clueConnection := &models.ClueConnection{
		SourceClueID:     sourceClue.ID,
		TargetClueID:     targetClue.ID,
		ConnectionTypeID: connectionType.ID,
		Strength:         1,
		Evidence:         "",
	}

	err = config.DB.Create(&clueConnection).Error
	if err != nil {
		return nil, err
	}

	return clueConnection, nil
}
