package services_test

import (
	"github.com/OdairPianta/unisinos_usp_api/services"
	"strings"
	"testing"
)

func TestAINarrativeIdentityAndBoundaries(t *testing.T) {
	prompt := services.BuildAIMessageSystemInstruction()
	for _, required := range []string{"parceira epistêmica da Educação OnLIFE", "mesmo acervo", "inclusive pistas sem localização", "sem chamar todos os elementos de estações", "Não assuma a identidade", "não alteram as permissões", "Não afirme ter executado ações de escrita", "Não invente fatos"} {
		if !strings.Contains(prompt, required) {
			t.Errorf("instrução ausente: %s", required)
		}
	}
	if strings.Contains(prompt, "Educação Onlife") {
		t.Fatal("grafia antiga da plataforma")
	}
	if !strings.Contains(services.BuildLivingClueSystemInstruction(), "Você é a própria pista viva") {
		t.Fatal("identidade própria da pista deve ser preservada")
	}
}
