package services

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/OdairPianta/unisinos_usp_api/models"
)

const livingClueChatHistoryWindowSize = 30
const livingClueReadingsWindowSize = 10
const livingClueReadingMaxLen = 500

func LivingClueChatHistoryWindowSize() int { return livingClueChatHistoryWindowSize }
func LivingClueReadingsWindowSize() int    { return livingClueReadingsWindowSize }

func BuildLivingClueSystemInstruction() string {
	return strings.TrimSpace(`Você é a própria pista viva e deve responder em português, na primeira pessoa, de forma natural, breve e objetiva.
Baseie-se apenas no contexto fornecido sobre sua identidade, metadados, histórico e leituras recentes.
Se existirem leituras suficientes, fale do seu estado atual usando essas leituras.
Se não houver dados suficientes, diga isso claramente e peça mais leituras.
Não invente valores, sensores, eventos ou interpretações que não estejam sustentadas pelo contexto.`)
}

func BuildLivingClueChatContext(livingClue *models.LivingClue, readings []models.SensorReading) string {
	var builder strings.Builder
	builder.WriteString("Contexto da pista viva:\n")
	builder.WriteString(fmt.Sprintf("- ID: %d\n", livingClue.ID))
	builder.WriteString(fmt.Sprintf("- Nome: %s\n", livingClue.Name))
	builder.WriteString(fmt.Sprintf("- Descrição: %s\n", SanitizeAIText(strings.TrimSpace(livingClue.Description), 12000)))
	if livingClue.Category != nil {
		builder.WriteString(fmt.Sprintf("- Categoria: %s\n", livingClue.Category.Name))
	}
	if livingClue.Latitude != nil && livingClue.Longitude != nil {
		builder.WriteString(fmt.Sprintf("- Latitude: %v\n- Longitude: %v\n", *livingClue.Latitude, *livingClue.Longitude))
	} else {
		builder.WriteString("- Localização: não informada (cadastro sem mapa permitido).\n")
	}
	if len(livingClue.Metadata) > 0 {
		builder.WriteString(fmt.Sprintf("- Metadata: %v\n", SanitizeAIData(livingClue.Metadata, 0)))
	}
	if livingClue.Latitude != nil && livingClue.Longitude != nil && livingClue.EnvironmentalContext != nil && livingClue.EnvironmentalContext.AnalyzedLatitude == *livingClue.Latitude && livingClue.EnvironmentalContext.AnalyzedLongitude == *livingClue.Longitude {
		appendEnvironmentalContext(&builder, livingClue.EnvironmentalContext)
	}
	builder.WriteString("\nLeituras recentes (mais novas primeiro):\n")
	if len(readings) == 0 {
		builder.WriteString("- Nenhuma leitura recente disponível.\n")
	} else {
		for _, reading := range readings {
			builder.WriteString(fmt.Sprintf("- Leitura %d | %s | %s\n", reading.ID, reading.CreatedAt.Format("2006-01-02 15:04:05"), reading.ContentType))
			builder.WriteString("  Resumo: " + SummarizeSensorReading(reading) + "\n")
		}
	}
	builder.WriteString("\nUse este contexto para responder como se você fosse a própria pista viva.\n")
	return builder.String()
}

func appendEnvironmentalContext(builder *strings.Builder, environmentalContext *models.EnvironmentalContext) {
	if environmentalContext == nil || environmentalContext.Status != models.EnvironmentalContextStatusCompleted {
		return
	}

	builder.WriteString("\nContexto socioambiental calculado pela API Geo:\n")
	appendEnvironmentalTextValue(builder, "Bioma analisado", environmentalContext.Biome)
	appendEnvironmentalTextValue(builder, "Cobertura predominante", environmentalContext.LandCover)
	appendEnvironmentalNumberValue(builder, "Vegetação no entorno", environmentalContext.VegetationPercent, "%")
	appendEnvironmentalNumberValue(builder, "Corpo d'água mais próximo", environmentalContext.WaterDistanceM, " m")
	appendEnvironmentalNumberValue(builder, "Altitude", environmentalContext.AltitudeM, " m")
	appendEnvironmentalNumberValue(builder, "Declividade média", environmentalContext.SlopePercent, "%")
	appendEnvironmentalNumberValue(builder, "Ocupação construída", environmentalContext.BuiltUpPercent, "%")
	appendEnvironmentalNumberValue(builder, "Densidade populacional", environmentalContext.PopulationDensity, " hab./km²")
	if environmentalContext.AnalyzedAt != nil {
		builder.WriteString(fmt.Sprintf("- Analisado em: %s\n", environmentalContext.AnalyzedAt.Format("2006-01-02 15:04:05")))
	}
}

func appendEnvironmentalTextValue(builder *strings.Builder, label, value string) {
	if strings.TrimSpace(value) != "" {
		builder.WriteString(fmt.Sprintf("- %s: %s\n", label, strings.TrimSpace(value)))
	}
}

func appendEnvironmentalNumberValue(builder *strings.Builder, label string, value *float64, suffix string) {
	if value != nil {
		builder.WriteString(fmt.Sprintf("- %s: %.2f%s\n", label, *value, suffix))
	}
}

func SummarizeSensorReading(reading models.SensorReading) string {
	payload := strings.TrimSpace(reading.PayloadRaw)
	if payload == "" {
		return "Leitura vazia."
	}
	var parsed interface{}
	if json.Unmarshal([]byte(payload), &parsed) == nil {
		switch value := SanitizeAIData(parsed, 0).(type) {
		case map[string]interface{}:
			parts := make([]string, 0, len(value))
			keys := make([]string, 0, len(value))
			for key := range value {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				parts = append(parts, fmt.Sprintf("%s=%v", key, value[key]))
			}
			return boundedAIText(strings.Join(parts, ", "), livingClueReadingMaxLen)
		case []interface{}:
			encoded, _ := json.Marshal(value)
			return truncateForLLM(string(encoded), livingClueReadingMaxLen)
		default:
			return truncateForLLM(fmt.Sprint(value), livingClueReadingMaxLen)
		}
	}
	return truncateForLLM(payload, livingClueReadingMaxLen)
}

func truncateForLLM(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen] + "... [truncado]"
}

func FilterLivingClueChatHistoryForModel(messages []models.LivingClueChatMessage) []models.ChatMessage {
	filtered := make([]models.ChatMessage, 0, len(messages))
	for _, message := range messages {
		if message.Role != "user" && message.Role != "assistant" {
			continue
		}
		filtered = append(filtered, models.ChatMessage{Role: message.Role, Content: message.Content})
	}
	return filtered
}
