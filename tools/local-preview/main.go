// Local preview of a restored server snapshot. Background jobs are deliberately
// omitted so queued production jobs are not executed during manual testing.
package main

import (
	"log"
	"os"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/routes"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Fatal(err)
	}
	if os.Getenv("DB_HOST") != "127.0.0.1" || os.Getenv("DB_NAME") != "rieonlife_server_copy" {
		log.Fatal("Local preview requires DB_HOST=127.0.0.1 and DB_NAME=rieonlife_server_copy")
	}
	config.InitDatabase()
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	routes.InitRoutes(router)
	if err := router.Run("127.0.0.1:8080"); err != nil {
		log.Fatal(err)
	}
}
