// Isolated browser validation. No jobs, mail, server data or user histories.
package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/enums"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/routes"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	if os.Getenv("DB_HOST") != "127.0.0.1" || !strings.HasPrefix(os.Getenv("DB_NAME"), "onlife_validation_") || os.Getenv("DB_PORT") != "33419" {
		log.Fatal("Integration preview requires the isolated loopback validation database")
	}
	password := os.Getenv("ONLIFE_PREVIEW_PASSWORD")
	if len(password) < 20 {
		log.Fatal("Set a generated synthetic preview credential")
	}
	config.InitDatabase()
	mode := "inventor"
	var user models.User
	result := config.DB.Where("email = ?", "integrations-preview@example.test").First(&user)
	if result.Error != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Fatal("Cannot prepare fixture account")
		}
		verifiedAt := time.Now()
		user = models.User{Name: "Validação de Integrações", Email: "integrations-preview@example.test", EmailVerifiedAt: &verifiedAt, Password: string(hash), Profile: enums.UserProfileEnumAdministrator, UsageMode: &mode}
		if config.DB.Create(&user).Error != nil {
			log.Fatal("Cannot create synthetic preview account")
		}
	}
	// /tmp credentials may rotate after a host restart while the isolated DB
	// persists. Reconcile only the fixed synthetic account in this guarded DB.
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil || config.DB.Model(&user).UpdateColumn("password", string(hash)).Error != nil {
			log.Fatal("Cannot reconcile synthetic preview credential")
		}
	}
	category := models.Category{Name: "Laboratório de integrações", Type: "experiencia", ColorHex: "#6ADCD1", MetadataTags: "observacao", Description: "Categoria sintética para validar integrações; não pertence a uma conta real."}
	if config.DB.Where("name = ?", category.Name).FirstOrCreate(&category).Error != nil {
		log.Fatal("Cannot prepare fixture category")
	}
	clue := models.LivingClue{Name: "Jardim Aurora — demonstração", CategoryID: category.ID, Description: "Esta pista é fictícia. O jardim possui 18 árvores, observadas em uma atividade de educação ambiental. Pode ser explorada sem coordenadas.", Metadata: models.JSONMap{"observacao": "Massa sintética; nenhuma atividade SUSI ou conta real."}, IsActive: true}
	if config.DB.Where("name = ?", clue.Name).FirstOrCreate(&clue).Error != nil {
		log.Fatal("Cannot prepare fixture clue")
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	routes.InitRoutes(router)
	log.Print("Synthetic integration preview ready on loopback :8082; background jobs disabled")
	if err := router.Run("127.0.0.1:8082"); err != nil {
		log.Fatal("Cannot start isolated preview")
	}
}
