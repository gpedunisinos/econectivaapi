package main

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/jobs"
	"github.com/OdairPianta/unisinos_usp_api/routes"
	"github.com/OdairPianta/unisinos_usp_api/services"
)

// swagger embed files

// @title           Educação Onlife
// @version         0.7.4
// @description     API da plataforma Educação Onlife

// @contact.name   API Support
// @contact.url    https://spotec.app/contato/
// @contact.email  contact@spotec.app

// @host      api.uniusp.gpedu.com.br
// @BasePath  /api

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
func main() {
	if err := services.SetupLogging(); err != nil {
		panic("Failed to setup logging: " + err.Error())
	}
	config.InitDatabase()
	go jobs.StartClient()
	services.InitSentry()
	routes.HandleRequest()
}
