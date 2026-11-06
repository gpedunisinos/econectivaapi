package services

import (
	"github.com/OdairPianta/unisinos_usp_api/models"
	"testing"
)

func TestNormalizeClueNames(t *testing.T) {
	for _, pair := range [][2]string{{"  SÃO   José ", "sao jose"}, {"GPe-dU", "gpe du"}, {"João da Silva", "joao da silva"}} {
		if got := NormalizeClueName(pair[0]); got != pair[1] {
			t.Fatalf("got %q want %q", got, pair[1])
		}
	}
	invalid := []string{"x"}
	if ValidateClueAliases(&invalid) == nil {
		t.Fatal("short alias accepted")
	}
	if ValidateClueAliases(nil) != nil {
		t.Fatal("omission must preserve compatibility")
	}
}

func TestDiscoveryNetworkIgnoresCVBoilerplateAndCatalogueWideTerms(t *testing.T) {
	source := models.LivingClue{Name: "Pessoa", Description: "Currículo atualizado, formação e sustentabilidade. Jogos e robótica."}
	target := models.LivingClue{Name: "Oficina", Description: "Formação, currículo e sustentabilidade. Robótica e jogos."}
	catalog := []models.LivingClue{}
	for i := 0; i < 10; i++ {
		catalog = append(catalog, models.LivingClue{Description: "Sustentabilidade, currículo atualizado e formação."})
	}
	words := discoverySourceWords(source, catalog, nil)
	if words["curriculo"] || words["atualizado"] || words["formacao"] || words["sustentabilidade"] {
		t.Fatal("boilerplate remained a network signal")
	}
	terms := sharedDiscoveryTerms(words, target)
	if len(terms) != 2 {
		t.Fatalf("expected two specific terms, got %v", terms)
	}
}
