package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/policies"
	"github.com/OdairPianta/unisinos_usp_api/repository"
	"github.com/OdairPianta/unisinos_usp_api/services"
)

var sourceFetcher = services.NewSourceFetcher()
var errAccess = errors.New("registro não encontrado ou consulta não autorizada")
var errQuery = errors.New("não foi possível realizar esta consulta agora")

type DatabaseBackend struct {
	userID       uint
	pinnedClueID uint
	Sources      *services.SourceFetcher
}

func NewDatabaseBackend(user models.User, pinnedClueID uint) *DatabaseBackend {
	return &DatabaseBackend{userID: user.ID, pinnedClueID: pinnedClueID, Sources: sourceFetcher}
}

func number(args map[string]interface{}, key string, fallback float64) (float64, error) {
	v, exists := args[key]
	if !exists {
		return fallback, nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return 0, errors.New("argumento inválido")
	}
	var value float64
	if json.Unmarshal(raw, &value) != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, errors.New("argumento inválido")
	}
	return value, nil
}
func uintArg(args map[string]interface{}, key string) (uint, error) {
	v, err := number(args, key, 0)
	if err != nil || v < 1 || v > 2147483647 || math.Trunc(v) != v {
		return 0, errors.New("ID inválido")
	}
	return uint(v), nil
}
func pageArgs(args map[string]interface{}) (int, int, error) {
	limit, err := number(args, "limit", 10)
	if err != nil || limit < 1 || limit > 20 || math.Trunc(limit) != limit {
		return 0, 0, errors.New("limite inválido")
	}
	offset, err := number(args, "offset", 0)
	if err != nil || offset < 0 || offset > 1000 || math.Trunc(offset) != offset {
		return 0, 0, errors.New("paginação inválida")
	}
	return int(limit), int(offset), nil
}
func clip(value string, limit int) string {
	r := []rune(value)
	if len(r) > limit {
		return string(r[:limit]) + "… [limitado]"
	}
	return value
}

func (b *DatabaseBackend) user(ctx context.Context) (*models.User, error) {
	if b.userID == 0 || config.DB == nil {
		return nil, errAccess
	}
	user := &models.User{}
	if config.DB.WithContext(ctx).First(user, b.userID).Error != nil {
		return nil, errAccess
	}
	return user, nil
}
func (b *DatabaseBackend) clue(ctx context.Context, user *models.User, id uint) (*models.LivingClue, error) {
	if b.pinnedClueID > 0 && id != b.pinnedClueID {
		return nil, errAccess
	}
	clue, err := repository.FindAIClue(ctx, id)
	if err != nil || !policies.NewLivingCluePolicy(user).View(clue) {
		return nil, errAccess
	}
	return clue, nil
}

func clueReference(c *models.LivingClue) map[string]interface{} {
	result := map[string]interface{}{"id": c.ID, "name": services.SanitizeAIText(c.Name, 255), "summary": services.SanitizeAIText(c.Description, 700), "link": fmt.Sprintf("#/space/research/%d", c.ID), "has_location": c.Latitude != nil && c.Longitude != nil}
	if c.Category != nil {
		result["category"] = map[string]interface{}{"id": c.Category.ID, "name": c.Category.Name}
	}
	return result
}
func nodeReference(n *models.Node) map[string]interface{} {
	result := map[string]interface{}{"id": n.ID, "name": services.SanitizeAIText(n.Name, 255), "color": n.Color, "type_id": n.NodeTypeID, "link": fmt.Sprintf("#/nodes/graph?focus=%d", n.ID)}
	if n.NodeType != nil {
		result["type"] = n.NodeType.Name
	}
	return result
}

func (b *DatabaseBackend) Execute(ctx context.Context, name string, args map[string]interface{}) (map[string]interface{}, error) {
	user, err := b.user(ctx)
	if err != nil {
		return nil, err
	}
	limit, offset, err := pageArgs(args)
	if err != nil {
		return nil, err
	}
	search, _ := args["text"].(string)
	if len([]rune(search)) > 200 {
		return nil, errors.New("busca muito longa")
	}
	switch name {
	case "pistas_pesquisar":
		if !policies.NewLivingCluePolicy(user).ViewAny() {
			return nil, errAccess
		}
		var category *uint
		if _, ok := args["category_id"]; ok {
			value, err := uintArg(args, "category_id")
			if err != nil {
				return nil, err
			}
			category = &value
		}
		var bounds *repository.ClueBounds
		count := 0
		for _, key := range []string{"north", "south", "east", "west"} {
			if _, ok := args[key]; ok {
				count++
			}
		}
		if count > 0 {
			if count != 4 {
				return nil, errors.New("informe os quatro limites geográficos")
			}
			n, e1 := number(args, "north", 0)
			s, e2 := number(args, "south", 0)
			e, e3 := number(args, "east", 0)
			w, e4 := number(args, "west", 0)
			if e1 != nil || e2 != nil || e3 != nil || e4 != nil || n > 90 || s < -90 || n < s || e > 180 || w < -180 || e < w {
				return nil, errors.New("limites geográficos inválidos")
			}
			bounds = &repository.ClueBounds{North: n, South: s, East: e, West: w}
		}
		clues, err := repository.FindAIClues(ctx, search, category, bounds, limit, offset)
		if err != nil {
			return nil, errQuery
		}
		rows := []interface{}{}
		for i := range clues {
			if policies.NewLivingCluePolicy(user).View(&clues[i]) {
				rows = append(rows, clueReference(&clues[i]))
			}
		}
		return map[string]interface{}{"records": rows, "limit": limit, "offset": offset, "geographic_filter": bounds != nil}, nil
	case "pista_obter", "pista_relacoes_listar", "pista_fontes_listar", "pista_fonte_consultar":
		id, err := uintArg(args, "id")
		if err != nil {
			return nil, err
		}
		clue, err := b.clue(ctx, user, id)
		if err != nil {
			return nil, err
		}
		if name == "pista_fontes_listar" {
			return map[string]interface{}{"clue_id": id, "sources": services.ClueSources(clue), "notice": "Referências e prévias não equivalem à leitura das fontes."}, nil
		}
		if name == "pista_fonte_consultar" {
			requested, _ := args["source_id"].(string)
			for _, source := range services.ClueSources(clue) {
				if source.ID != requested {
					continue
				}
				if !source.Consultable {
					return map[string]interface{}{"clue_id": id, "source_id": requested, "status": "blocked", "limitation": source.Limitation}, nil
				}
				if source.Provider == "jitsi" {
					return map[string]interface{}{"clue_id": id, "source_id": requested, "url": source.URL, "status": "interactive_only", "limitation": "Referência a um encontro. A IA não acessou nem acompanhou a reunião, sua presença, mensagens, áudio ou vídeo. O link não fornece uma transcrição."}, nil
				}
				if source.Provider != "" {
					return map[string]interface{}{"clue_id": id, "source_id": requested, "url": source.URL, "status": "interactive_only", "limitation": "Referência a conteúdo interativo ou vídeo. Não foi assistido, transcrito ou explorado pela IA; use somente o texto autoral da pista e apresente este link como acesso externo."}, nil
				}
				return map[string]interface{}{"clue_id": id, "source_id": requested, "source": b.Sources.Fetch(ctx, source.URL)}, nil
			}
			return nil, errors.New("a fonte não pertence à pista autorizada")
		}
		if name == "pista_relacoes_listar" {
			connections, err := repository.FindAIClueConnections(ctx, id, limit, offset)
			if err != nil {
				return nil, errQuery
			}
			rows := []interface{}{}
			for _, link := range connections {
				if link.SourceClue == nil || link.TargetClue == nil || !policies.NewLivingCluePolicy(user).View(link.SourceClue) || !policies.NewLivingCluePolicy(user).View(link.TargetClue) {
					continue
				}
				row := map[string]interface{}{"id": link.ID, "source": clueReference(link.SourceClue), "target": clueReference(link.TargetClue), "evidence": clip(link.Evidence, 1500)}
				if link.ConnectionType != nil {
					row["type"] = link.ConnectionType.Name
				}
				rows = append(rows, row)
			}
			return map[string]interface{}{"clue_id": id, "relations": rows, "limit": limit, "offset": offset}, nil
		}
		result := clueReference(clue)
		result["description"] = services.SanitizeAIText(clue.Description, 12000)
		result["metadata"] = services.SanitizeAIData(clue.Metadata, 0)
		result["sources"] = services.ClueSources(clue)
		result["updated_at"] = clue.UpdatedAt
		if clue.Latitude != nil && clue.Longitude != nil && *clue.Latitude >= -90 && *clue.Latitude <= 90 && *clue.Longitude >= -180 && *clue.Longitude <= 180 {
			result["location"] = map[string]interface{}{"latitude": *clue.Latitude, "longitude": *clue.Longitude}
			env := clue.EnvironmentalContext
			if env != nil && env.Status == models.EnvironmentalContextStatusCompleted && env.AnalyzedLatitude == *clue.Latitude && env.AnalyzedLongitude == *clue.Longitude {
				result["environment"] = map[string]interface{}{"biome": env.Biome, "land_cover": env.LandCover, "vegetation_percent": env.VegetationPercent, "water_distance_m": env.WaterDistanceM, "altitude_m": env.AltitudeM, "slope_percent": env.SlopePercent, "built_up_percent": env.BuiltUpPercent, "population_density_km2": env.PopulationDensity, "analyzed_at": env.AnalyzedAt}
			}
		} else {
			result["has_location"] = false
			result["location_notice"] = "Localização não informada; não inferir bioma, distância ou posição física."
		}
		files := []interface{}{}
		for _, file := range clue.Files {
			if len(files) >= 20 {
				break
			}
			row := map[string]interface{}{"name": clip(file.Name, 255)}
			if _, err := services.SafeSourceURL(file.PublicURL); err == nil {
				row["url"] = file.PublicURL
			}
			files = append(files, row)
		}
		result["attachments"] = files
		readings := []interface{}{}
		for _, reading := range clue.SensorReadings {
			readings = append(readings, map[string]interface{}{"id": reading.ID, "created_at": reading.CreatedAt, "summary": clip(services.SummarizeSensorReading(reading), 1500)})
		}
		result["readings"] = readings
		result["notice"] = "Dados persistidos atuais; relações completas devem ser consultadas com pista_relacoes_listar. Fontes listadas não foram lidas automaticamente."
		return result, nil
	case "problematicas_pesquisar":
		if !policies.NewNodePolicy(user).ViewAny() {
			return nil, errAccess
		}
		var nodeType *uint
		if _, ok := args["type_id"]; ok {
			value, err := uintArg(args, "type_id")
			if err != nil {
				return nil, err
			}
			nodeType = &value
		}
		nodes, err := repository.FindAINodes(ctx, search, nodeType, limit, offset)
		if err != nil {
			return nil, errQuery
		}
		rows := []interface{}{}
		for i := range nodes {
			if policies.NewNodePolicy(user).View(&nodes[i]) {
				rows = append(rows, nodeReference(&nodes[i]))
			}
		}
		return map[string]interface{}{"records": rows, "limit": limit, "offset": offset}, nil
	case "problematica_obter":
		id, err := uintArg(args, "id")
		if err != nil {
			return nil, err
		}
		node, links, err := repository.FindAINode(ctx, id)
		if err != nil || !policies.NewNodePolicy(user).View(node) {
			return nil, errAccess
		}
		result := nodeReference(node)
		connections := []interface{}{}
		for _, link := range links {
			if link.SourceNode == nil || link.TargetNode == nil || !policies.NewNodePolicy(user).View(link.SourceNode) || !policies.NewNodePolicy(user).View(link.TargetNode) {
				continue
			}
			connections = append(connections, map[string]interface{}{"id": link.ID, "source": nodeReference(link.SourceNode), "target": nodeReference(link.TargetNode)})
		}
		result["connections"] = connections
		result["connections_limit"] = 20
		return result, nil
	default:
		return nil, errors.New("ferramenta não disponível; somente consultas declaradas são permitidas")
	}
}
