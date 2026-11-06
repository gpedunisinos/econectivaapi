package tests

import (
	"bytes"
	"context"
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
	"testing"
)

func TestRieonlifeAccountAndParticipationFlow(t *testing.T) {
	setupDatabase()
	r := routesSetup()
	u, token := initUser()
	_, other := initUser()
	request := func(auth, method, path string, body interface{}, status int) []byte {
		t.Helper()
		data, _ := json.Marshal(body)
		req := httptest.NewRequest(method, "/api"+path, bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+auth)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, status, w.Code, w.Body.String())
		return w.Body.Bytes()
	}
	request(token, "PUT", "/me/preferences", map[string]string{"usage_mode": "inventor"}, 200)
	require.Contains(t, string(request(token, "GET", "/me/preferences", nil, 200)), "inventor")
	request(token, "PUT", "/me/preferences", map[string]string{"usage_mode": "admin"}, 400)
	cat, err := factories.CreateCategory()
	require.NoError(t, err)
	body := map[string]interface{}{"name": "Vivência não geográfica ✨", "category_id": cat.ID, "description": "forged plain text", "description_delta": `[{"insert":"Olá 🌱\n"},{"insert":{"video":"https://youtu.be/dQw4w9WgXcQ"}},{"insert":"\n"}]`}
	var clue models.LivingClue
	require.NoError(t, json.Unmarshal(request(token, "POST", "/living_clues", body, 200), &clue))
	require.Nil(t, clue.Latitude)
	require.Nil(t, clue.Longitude)
	require.Equal(t, "Olá 🌱", clue.Description)
	var count int64
	config.DB.Model(&models.Job{}).Where("payload LIKE ?", fmt.Sprintf("%%\"living_clue_id\":%d%%", clue.ID)).Count(&count)
	require.Zero(t, count)
	path := fmt.Sprintf("/living_clues/%d", clue.ID)
	locker := fmt.Sprintf("/me/locker/living_clue/%d", clue.ID)
	request(token, "PUT", locker, nil, 200)
	request(token, "PUT", locker, nil, 200)
	var items []models.LockerItem
	require.NoError(t, json.Unmarshal(request(token, "GET", "/me/locker", nil, 200), &items))
	require.Len(t, items, 1)
	require.Equal(t, "[]", string(request(other, "GET", "/me/locker", nil, 200)))
	// Saved agents share the account boundary without copying instructions.
	agent := models.ChatTemplate{Name: "Agente de referência", SystemInstruction: "private configuration", UserID: u.ID}
	require.NoError(t, config.DB.Create(&agent).Error)
	agentLocker := fmt.Sprintf("/me/locker/agent/%d", agent.ID)
	request(token, "PUT", agentLocker, nil, 200)
	require.NotContains(t, string(request(token, "GET", "/me/locker", nil, 200)), agent.SystemInstruction)
	require.Equal(t, "[]", string(request(other, "GET", "/me/locker", nil, 200)))
	request(token, "DELETE", agentLocker, nil, 200)
	request(token, "PUT", "/me/locker/agent/0", nil, 400)
	request(token, "PUT", "/me/locker/agent/4294967295", nil, 404)
	// Contributions derive from edit records and cannot be selected for another account.
	edit := models.ClueEdit{LivingClueID: clue.ID, UserID: u.ID}
	require.NoError(t, config.DB.Create(&edit).Error)
	contributions := string(request(token, "GET", "/me/contributions?user_id=0", nil, 200))
	require.Contains(t, contributions, clue.Name)
	require.NotContains(t, contributions, "description_delta")
	require.Equal(t, "[]", string(request(other, "GET", "/me/contributions", nil, 200)))
	require.NoError(t, config.DB.Delete(&edit).Error)
	// Add the valid zero pair, omit to preserve it, then explicitly remove.
	body["latitude"] = 0
	body["longitude"] = 0
	require.NoError(t, json.Unmarshal(request(token, "PUT", path, body, 200), &clue))
	require.NotNil(t, clue.Latitude)
	require.Zero(t, *clue.Latitude)
	oldContext, err := repository.FindEnvironmentalContextByLivingClueID(clue.ID)
	require.NoError(t, err)
	delete(body, "latitude")
	delete(body, "longitude")
	require.NoError(t, json.Unmarshal(request(token, "PUT", path, body, 200), &clue))
	require.NotNil(t, clue.Latitude)
	body["remove_location"] = true
	require.NoError(t, json.Unmarshal(request(token, "PUT", path, body, 200), &clue))
	require.Nil(t, clue.Latitude)
	oldContext.Status = models.EnvironmentalContextStatusCompleted
	require.NoError(t, repository.SaveEnvironmentalContext(oldContext))
	_, err = repository.FindEnvironmentalContextByLivingClueID(clue.ID)
	require.Error(t, err, "stale result must not recreate context")
	require.NoError(t, jobs.RegisterEnvironmentalContextJob(&clue))
	body["remove_location"] = false
	body["latitude"] = 1
	request(token, "PUT", path, body, 400)
	delete(body, "latitude")
	var edits []models.ClueEdit
	require.NoError(t, json.Unmarshal(request(token, "GET", path+"/edits", nil, 200), &edits))
	require.Len(t, edits, 4)
	require.Equal(t, u.ID, edits[0].Author.ID)
	// Explorer can participate and bookmark, but cannot author platform content.
	request(token, "PUT", "/me/preferences", map[string]string{"usage_mode": "explorer"}, 200)
	request(token, "PUT", path, body, 403)
	var root, reply, reply2 models.ClueComment
	require.NoError(t, json.Unmarshal(request(token, "POST", path+"/comments", map[string]string{"content": "Uma descoberta 🌱\nSegunda linha"}, 200), &root))
	require.NoError(t, json.Unmarshal(request(other, "POST", path+"/comments", map[string]interface{}{"content": "Resposta ✨", "reply_to_id": root.ID}, 200), &reply))
	require.NoError(t, json.Unmarshal(request(token, "POST", path+"/comments", map[string]interface{}{"content": "Respondendo à resposta", "reply_to_id": reply.ID}, 200), &reply2))
	require.Equal(t, root.ID, *reply2.RootID)
	require.Equal(t, reply.ID, *reply2.ReplyToID)
	request(other, "PUT", fmt.Sprintf("%s/comments/%d", path, root.ID), map[string]string{"content": "forged"}, 403)
	request(token, "POST", path+"/comments", map[string]string{"content": " \n "}, 400)
	request(token, "PUT", fmt.Sprintf("%s/comments/%d", path, root.ID), map[string]string{"content": "Editado 🌱"}, 200)
	request(token, "DELETE", fmt.Sprintf("%s/comments/%d", path, root.ID), nil, 200)
	var messages []models.ClueComment
	require.NoError(t, json.Unmarshal(request(token, "GET", path+"/comments", nil, 200), &messages))
	require.Len(t, messages, 1)
	require.Empty(t, messages[0].Content)
	require.NotNil(t, messages[0].RemovedAt)
	require.NoError(t, json.Unmarshal(request(token, "GET", fmt.Sprintf("%s/comments?root_id=%d", path, root.ID), nil, 200), &messages))
	require.Len(t, messages, 2)
	require.Equal(t, reply.ID, messages[0].ID)
	require.NotNil(t, messages[0].ReplyToAuthor)
	require.Equal(t, u.Name, messages[0].ReplyToAuthor.Name)
	require.NotContains(t, string(request(token, "GET", path+"/comments", nil, 200)), "email")
	request(token, "DELETE", locker, nil, 200)
	require.Equal(t, "[]", string(request(token, "GET", "/me/locker", nil, 200)))
	require.NoError(t, config.DB.First(&clue, clue.ID).Error)
}

func TestRieonlifeVideoValidation(t *testing.T) {
	for _, url := range []string{"https://youtu.be/dQw4w9WgXcQ", "https://www.youtube.com/watch?v=dQw4w9WgXcQ", "https://www.youtube.com/shorts/dQw4w9WgXcQ"} {
		require.NotEmpty(t, services.YouTubeID(url))
	}
	for _, url := range []string{"https://youtube.com.evil.test/watch?v=dQw4w9WgXcQ", "javascript:alert(1)", "https://youtube.com/playlist?list=test", "https://youtu.be/nope"} {
		require.Empty(t, services.YouTubeID(url))
	}
	require.NoError(t, services.ValidateQuillDelta(`[{"insert":{"video":"https://youtu.be/dQw4w9WgXcQ"}},{"insert":"\n"}]`))
	require.Error(t, services.ValidateQuillDelta(`[{"insert":{"video":"https://evil.test/movie"}}]`))
}

func TestRieonlifeEnvironmentalJobTracksCurrentLocation(t *testing.T) {
	setupDatabase()
	clue, err := factories.CreateLivingClue()
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"biome":"Mata Atlântica"}`))
	}))
	defer server.Close()
	t.Setenv("ENVIRONMENTAL_CONTEXT_API_URL", server.URL)
	require.NoError(t, jobs.RegisterEnvironmentalContextJob(clue))
	payload := fmt.Sprintf(`{"living_clue_id":%d}`, clue.ID)
	require.NoError(t, jobs.RunEnvironmentalContextJob(payload))
	ctx, err := repository.FindEnvironmentalContextByLivingClueID(clue.ID)
	require.NoError(t, err)
	require.Equal(t, models.EnvironmentalContextStatusCompleted, ctx.Status)
	require.Equal(t, "Mata Atlântica", ctx.Biome)
	stale := *ctx
	clue.Latitude = testCoordinate(20)
	require.NoError(t, config.DB.Model(clue).Update("latitude", 20).Error)
	require.NoError(t, repository.SaveEnvironmentalContext(&stale))
	// Moving away and back must not revive a previous generation.
	require.NoError(t, jobs.RegisterEnvironmentalContextJob(clue))
	newctx, err := repository.FindEnvironmentalContextByLivingClueID(clue.ID)
	require.NoError(t, err)
	require.Equal(t, models.EnvironmentalContextStatusPending, newctx.Status)
	require.NoError(t, jobs.RunEnvironmentalContextJob(payload))
	newctx, err = repository.FindEnvironmentalContextByLivingClueID(clue.ID)
	require.NoError(t, err)
	require.Equal(t, models.EnvironmentalContextStatusCompleted, newctx.Status)
	require.Equal(t, 20.0, newctx.AnalyzedLatitude)
}

func TestRieonlifePreviewRefusesPrivateDestinations(t *testing.T) {
	setupDatabase()
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++; _, _ = w.Write([]byte("<title>private</title>")) }))
	defer server.Close()
	_, err := services.GetPublicLinkPreview(context.Background(), server.URL)
	require.Error(t, err)
	require.Zero(t, hits)
	_, err = services.GetPublicLinkPreview(context.Background(), "file:///etc/passwd")
	require.Error(t, err)
}

func TestRieonlifeCluePreviewReferencesFollowTheDocument(t *testing.T) {
	setupDatabase()
	user, token := initUser()
	category, err := factories.CreateCategory()
	require.NoError(t, err)
	delta := `[{"insert":"https://example.org/a#fragment https://example.org/a https://youtu.be/dQw4w9WgXcQ\n"},{"insert":"palavra","attributes":{"link":"https://example.org/b"}},{"insert":"\n"},{"insert":{"video":"https://www.youtube.com/watch?v=dQw4w9WgXcQ"}},{"insert":"\n"}]`
	clue := models.LivingClue{Name: "Prévias separadas", CategoryID: category.ID, DescriptionDelta: delta}
	require.NoError(t, repository.CreateLivingClue(&clue))
	var refs []models.CluePreviewReference
	require.NoError(t, config.DB.Where("living_clue_id = ?", clue.ID).Find(&refs).Error)
	require.Len(t, refs, 2)
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/living_clues/%d/link_previews", clue.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	routesSetup().ServeHTTP(response, req)
	require.Equal(t, 200, response.Code, response.Body.String())
	var previews []models.LinkPreview
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &previews))
	require.Len(t, previews, 2)

	updated := clue
	updated.DescriptionDelta = `[{"insert":"Sem links\n"}]`
	require.NoError(t, repository.UpdateLivingClueWithEdit(&clue, &updated, user.ID, false))
	var count int64
	config.DB.Model(&models.CluePreviewReference{}).Where("living_clue_id = ?", clue.ID).Count(&count)
	require.Zero(t, count)
}
