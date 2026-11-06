package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/jobs"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/repository"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/OdairPianta/unisinos_usp_api/tests/factories"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestClueDiscoveryAliasesAndGroundedSuggestions(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	user, token := initUser()
	request := func(method, path string, body interface{}, status int) []byte {
		t.Helper()
		raw, _ := json.Marshal(body)
		r := httptest.NewRequest(method, "/api"+path, bytes.NewReader(raw))
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		require.Equal(t, status, w.Code, w.Body.String())
		return w.Body.Bytes()
	}
	request("PUT", "/me/preferences", map[string]string{"usage_mode": "inventor"}, 200)
	category, err := factories.CreateCategory()
	require.NoError(t, err)
	names := []string{"Unisinos", "UNISINOS"}
	institution := models.LivingClue{Name: "Universidade do Vale do Rio dos Sinos", CategoryID: category.ID, IsActive: true, AliasNames: &names}
	require.NoError(t, repository.CreateLivingClue(&institution, user.ID))
	var aliases []models.ClueAlias
	require.NoError(t, config.DB.Where("living_clue_id = ?", institution.ID).Find(&aliases).Error)
	require.Len(t, aliases, 1)
	read, err := repository.FindLivingClueByID(fmt.Sprint(institution.ID))
	require.NoError(t, err)
	require.Len(t, read.Aliases, 1)
	// Omission preserves names for older clients; an explicit empty list clears.
	body := map[string]interface{}{"name": institution.Name, "category_id": category.ID, "description": "Instituição"}
	request("PUT", fmt.Sprintf("/living_clues/%d", institution.ID), body, 200)
	require.NoError(t, config.DB.Where("living_clue_id = ?", institution.ID).Find(&aliases).Error)
	require.Len(t, aliases, 1)
	body["alias_names"] = []string{}
	request("PUT", fmt.Sprintf("/living_clues/%d", institution.ID), body, 200)
	require.NoError(t, config.DB.Where("living_clue_id = ?", institution.ID).Find(&aliases).Error)
	require.Empty(t, aliases)
	body["alias_names"] = []string{"Unisinos"}
	request("PUT", fmt.Sprintf("/living_clues/%d", institution.ID), body, 200)
	body["alias_names"] = []string{"x"}
	request("PUT", fmt.Sprintf("/living_clues/%d", institution.ID), body, 400)
	source := models.LivingClue{Name: "Pessoa de teste", Description: "Estudei na Unisinos.", CategoryID: category.ID, IsActive: true}
	require.NoError(t, repository.CreateLivingClue(&source, user.ID))
	// Internal extractor supplies references, not links. Foreign/self/inactive
	// suggestions are filtered again by the API when the result is read.
	engine := httptest.NewServer(httpHandler(func(input services.DiscoveryRequest) services.DiscoveryResult {
		return services.DiscoveryResult{References: []services.DiscoveryReference{{TargetID: institution.ID, Mention: "Unisinos", Evidence: "Estudei na Unisinos."}, {TargetID: source.ID, Mention: "Pessoa de teste"}, {TargetID: 999999}}, AliasCandidates: []services.DiscoveryAliasCandidate{{Name: "Nome proposto", Evidence: "Autodefinição explícita."}}}
	}))
	defer engine.Close()
	t.Setenv("CLUE_DISCOVERY_URL", engine.URL)
	path := fmt.Sprintf("/living_clues/%d/connection_suggestions", source.ID)
	response := request("GET", path, nil, 200)
	require.Contains(t, string(response), "queued")
	process := func() {
		t.Helper()
		var pending []models.Job
		require.NoError(t, config.DB.Where("type = ?", services.ClueDiscoveryJob).Order("id").Find(&pending).Error)
		for _, job := range pending {
			require.NoError(t, jobs.ProcessJob(job))
			require.NoError(t, config.DB.Delete(&job).Error)
		}
	}
	process()
	var result struct {
		Status      string                  `json:"status"`
		Suggestions []models.ClueSuggestion `json:"suggestions"`
		Aliases     []models.ClueAlias      `json:"aliases"`
	}
	require.NoError(t, json.Unmarshal(request("GET", path, nil, 200), &result))
	require.Equal(t, "ready", result.Status)
	require.Len(t, result.Suggestions, 1)
	require.Equal(t, institution.ID, result.Suggestions[0].TargetClueID)
	var count int64
	config.DB.Model(&models.ClueConnection{}).Where("source_clue_id = ?", source.ID).Count(&count)
	require.Zero(t, count)
	require.Len(t, result.Aliases, 1)
	require.Equal(t, "pending", result.Aliases[0].Status)
	aliasID := result.Aliases[0].ID
	request("PUT", fmt.Sprintf("/living_clues/%d/aliases/%d", source.ID, aliasID), map[string]string{"status": "approved"}, 200)
	process()
	require.NoError(t, config.DB.First(&aliases, "living_clue_id = ? AND status = ?", source.ID, "approved").Error)
	// Dismissal persists through a refresh for the same content snapshot.
	require.NoError(t, json.Unmarshal(request("GET", path, nil, 200), &result))
	require.Len(t, result.Suggestions, 1)
	suggestion := result.Suggestions[0]
	request("POST", fmt.Sprintf("%s/%d/dismiss", path, suggestion.ID), nil, 200)
	request("POST", path+"/refresh", nil, 202)
	process()
	require.NoError(t, json.Unmarshal(request("GET", path, nil, 200), &result))
	require.Empty(t, result.Suggestions)
	connectionType, err := factories.CreateConnectionType()
	require.NoError(t, err)
	require.NoError(t, config.DB.Create(&models.ClueConnection{SourceClueID: source.ID, TargetClueID: institution.ID, ConnectionTypeID: connectionType.ID, Strength: 1}).Error)
	request("GET", path, nil, 200)
	process()
	require.NoError(t, json.Unmarshal(request("GET", path, nil, 200), &result))
	require.Empty(t, result.Suggestions)
	// Inactive/deleted candidates and existing links never leak back into results.
	require.NoError(t, config.DB.Model(&models.ClueSuggestion{}).Where("id = ?", suggestion.ID).Update("dismissed", false).Error)
	require.NoError(t, config.DB.Model(&models.LivingClue{}).Where("id = ?", institution.ID).Update("is_active", false).Error)
	request("GET", path, nil, 200)
	process()
	require.NoError(t, json.Unmarshal(request("GET", path, nil, 200), &result))
	require.Empty(t, result.Suggestions)
	// Explorers can inspect suggestions, but cannot confirm names or links.
	request("PUT", "/me/preferences", map[string]string{"usage_mode": "explorer"}, 200)
	request("POST", path+"/refresh", nil, 403)
	request("PUT", fmt.Sprintf("/living_clues/%d/aliases/%d", source.ID, aliasID), map[string]string{"status": "approved"}, 403)
}

func httpHandler(handle func(services.DiscoveryRequest) services.DiscoveryResult) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input services.DiscoveryRequest
		json.NewDecoder(r.Body).Decode(&input)
		json.NewEncoder(w).Encode(handle(input))
	})
}

func TestClueDiscoveryRealExtractor(t *testing.T) {
	url := os.Getenv("CLUE_DISCOVERY_TEST_URL")
	if url == "" {
		t.Skip("local extractor not requested")
	}
	setupDatabase()
	t.Setenv("CLUE_DISCOVERY_URL", url)
	category, err := factories.CreateCategory()
	require.NoError(t, err)
	names := []string{"Instituto Teste São José"}
	target := models.LivingClue{Name: "Instituto de Pesquisa de Demonstração", CategoryID: category.ID, IsActive: true, AliasNames: &names}
	require.NoError(t, repository.CreateLivingClue(&target))
	source := models.LivingClue{Name: "Atividade para teste", Description: "A atividade aconteceu no Instituto Teste Sao Jose.", CategoryID: category.ID, IsActive: true}
	require.NoError(t, repository.CreateLivingClue(&source))
	require.NoError(t, services.QueueClueDiscovery(source.ID, false))
	var job models.Job
	require.NoError(t, config.DB.Where("type = ?", services.ClueDiscoveryJob).Order("id DESC").First(&job).Error)
	require.NoError(t, jobs.ProcessJob(job))
	var state models.ClueDiscoveryState
	require.NoError(t, config.DB.First(&state, "living_clue_id = ?", source.ID).Error)
	require.Equal(t, "ready", state.Status)
	var suggestions []models.ClueSuggestion
	require.NoError(t, config.DB.Where("source_clue_id = ? AND target_clue_id = ?", source.ID, target.ID).Find(&suggestions).Error)
	require.Len(t, suggestions, 1)
	require.Contains(t, suggestions[0].Evidence, "Instituto Teste Sao Jose")
}

func TestClueDiscoveryStaleResponsesAndOutage(t *testing.T) {
	setupDatabase()
	category, err := factories.CreateCategory()
	require.NoError(t, err)
	source := models.LivingClue{Name: "Registro inicial", Description: "Texto anterior", CategoryID: category.ID, IsActive: true}
	require.NoError(t, repository.CreateLivingClue(&source))
	called := false
	engine := httptest.NewServer(httpHandler(func(input services.DiscoveryRequest) services.DiscoveryResult {
		if !called {
			called = true
			config.DB.Model(&models.LivingClue{}).Where("id = ?", source.ID).Update("description", "Texto atualizado durante a análise")
		}
		return services.DiscoveryResult{}
	}))
	defer engine.Close()
	t.Setenv("CLUE_DISCOVERY_URL", engine.URL)
	require.NoError(t, services.QueueClueDiscovery(source.ID, false))
	var job models.Job
	require.NoError(t, config.DB.Where("type = ?", services.ClueDiscoveryJob).Order("id DESC").First(&job).Error)
	require.NoError(t, jobs.ProcessJob(job))
	require.NoError(t, config.DB.Delete(&job).Error)
	var state models.ClueDiscoveryState
	require.NoError(t, config.DB.First(&state, "living_clue_id = ?", source.ID).Error)
	require.Equal(t, "queued", state.Status)
	var replacement models.Job
	require.NoError(t, config.DB.Where("type = ?", services.ClueDiscoveryJob).Order("id DESC").First(&replacement).Error)
	require.NotEqual(t, job.ID, replacement.ID)
	require.NoError(t, jobs.ProcessJob(replacement))
	config.DB.Delete(&replacement)
	require.NoError(t, config.DB.First(&state, "living_clue_id = ?", source.ID).Error)
	require.Equal(t, "ready", state.Status)
	t.Setenv("CLUE_DISCOVERY_URL", "http://127.0.0.1:1")
	require.NoError(t, services.QueueClueDiscovery(source.ID, true))
	replacement = models.Job{}
	require.NoError(t, config.DB.Where("type = ?", services.ClueDiscoveryJob).Order("id DESC").First(&replacement).Error)
	require.NoError(t, jobs.ProcessJob(replacement))
	config.DB.Delete(&replacement)
	require.NoError(t, config.DB.First(&state, "living_clue_id = ?", source.ID).Error)
	require.Equal(t, "failed", state.Status)
	var before, after int64
	config.DB.Model(&models.Job{}).Where("type = ?", services.ClueDiscoveryJob).Count(&before)
	require.NoError(t, services.QueueClueDiscovery(source.ID, false))
	config.DB.Model(&models.Job{}).Where("type = ?", services.ClueDiscoveryJob).Count(&after)
	require.Equal(t, before, after)
}
