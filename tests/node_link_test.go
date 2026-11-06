package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/tests/factories"
	"github.com/stretchr/testify/assert"
)

func TestNodeLinkFindAll(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateNodeLink()
	assert.Nil(t, err)

	request, _ := http.NewRequest("GET", "/api/node_links?search="+fmt.Sprint(model.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())
	var result []models.NodeLink
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Greater(t, len(result), 0, "Returned body: "+recorder.Body.String())
}

func TestNodeLinkCreate(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	sourceNode, err := factories.CreateNode()
	assert.Nil(t, err)
	targetNode, err := factories.CreateNode()
	assert.Nil(t, err)

	payload := map[string]interface{}{
		"source_node_id": sourceNode.ID,
		"target_node_id": targetNode.ID,
	}

	body, _ := json.Marshal(payload)

	request, _ := http.NewRequest("POST", "/api/node_links", bytes.NewBuffer(body))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.NodeLink
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, payload["source_node_id"], result.SourceNodeID)
	assert.Equal(t, payload["target_node_id"], result.TargetNodeID)
}

func TestNodeLinkFind(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateNodeLink()
	assert.Nil(t, err)

	request, _ := http.NewRequest("GET", "/api/node_links/"+fmt.Sprint(model.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.NodeLink
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Equal(t, model.ID, result.ID)
}

func TestNodeLinkUpdate(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateNodeLink()
	assert.Nil(t, err)

	newSourceNode, err := factories.CreateNode()
	assert.Nil(t, err)
	newTargetNode, err := factories.CreateNode()
	assert.Nil(t, err)

	payload := map[string]interface{}{
		"source_node_id": newSourceNode.ID,
		"target_node_id": newTargetNode.ID,
	}

	body, _ := json.Marshal(payload)

	request, _ := http.NewRequest("PUT", "/api/node_links/"+fmt.Sprint(model.ID), bytes.NewBuffer(body))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.NodeLink
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Equal(t, payload["source_node_id"], result.SourceNodeID)
	assert.Equal(t, payload["target_node_id"], result.TargetNodeID)
}

func TestNodeLinkDelete(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateNodeLink()
	assert.Nil(t, err)

	request, _ := http.NewRequest("DELETE", "/api/node_links/"+fmt.Sprint(model.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.NodeLink
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Equal(t, model.ID, result.ID)
}

func TestNodeLinkAvoidSelfLink(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	node, err := factories.CreateNode()
	assert.Nil(t, err)

	payload := map[string]interface{}{
		"source_node_id": node.ID,
		"target_node_id": node.ID,
	}
	body, _ := json.Marshal(payload)

	request, _ := http.NewRequest("POST", "/api/node_links", bytes.NewBuffer(body))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusBadRequest, recorder.Code, "Returned body: "+recorder.Body.String())
}

func TestNodeLinkAvoidDuplicates(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateNodeLink()
	assert.Nil(t, err)

	payload := map[string]interface{}{
		"source_node_id": model.SourceNodeID,
		"target_node_id": model.TargetNodeID,
	}
	body, _ := json.Marshal(payload)

	request, _ := http.NewRequest("POST", "/api/node_links", bytes.NewBuffer(body))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusBadRequest, recorder.Code, "Returned body: "+recorder.Body.String())
}

func TestNodeLinkCanBeRecreatedAfterDelete(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	sourceNode, err := factories.CreateNode()
	assert.Nil(t, err)
	targetNode, err := factories.CreateNode()
	assert.Nil(t, err)

	createPayload := map[string]interface{}{
		"source_node_id": sourceNode.ID,
		"target_node_id": targetNode.ID,
	}
	createBody, _ := json.Marshal(createPayload)

	createRequest, _ := http.NewRequest("POST", "/api/node_links", bytes.NewBuffer(createBody))
	createRequest.Header.Set("Authorization", "Bearer "+token)
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)
	assert.Equal(t, http.StatusOK, createRecorder.Code, "Returned body: "+createRecorder.Body.String())

	var created models.NodeLink
	_ = json.Unmarshal(createRecorder.Body.Bytes(), &created)

	deleteRequest, _ := http.NewRequest("DELETE", "/api/node_links/"+fmt.Sprint(created.ID), nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+token)
	deleteRecorder := httptest.NewRecorder()
	router.ServeHTTP(deleteRecorder, deleteRequest)
	assert.Equal(t, http.StatusOK, deleteRecorder.Code, "Returned body: "+deleteRecorder.Body.String())

	recreateRequest, _ := http.NewRequest("POST", "/api/node_links", bytes.NewBuffer(createBody))
	recreateRequest.Header.Set("Authorization", "Bearer "+token)
	recreateRecorder := httptest.NewRecorder()
	router.ServeHTTP(recreateRecorder, recreateRequest)
	assert.Equal(t, http.StatusOK, recreateRecorder.Code, "Returned body: "+recreateRecorder.Body.String())
}

func TestNodeLinkAvoidReverseDuplicates(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateNodeLink()
	assert.Nil(t, err)

	payload := map[string]interface{}{
		"source_node_id": model.TargetNodeID,
		"target_node_id": model.SourceNodeID,
	}
	body, _ := json.Marshal(payload)

	request, _ := http.NewRequest("POST", "/api/node_links", bytes.NewBuffer(body))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusBadRequest, recorder.Code, "Returned body: "+recorder.Body.String())
}
