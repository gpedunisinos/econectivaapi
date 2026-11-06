package services

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/stretchr/testify/require"
)

func TestCurrentContextUsesAccountModeAndExcludesSensitiveData(t *testing.T) {
	mode := "explorer"
	context := BuildCurrentScreenContext(&models.AIScreenContext{Version: 1, ScreenName: "edição", Route: "/living_clues/update?id=42", UsageMode: "inventor", UsageModeLabel: "Inventor", References: map[string]uint{"living_clue_id": 42, "user_id": 999}, FilledFields: map[string]string{"nome": "Rascunho", "password": "not-for-context"}, Draft: map[string]interface{}{"unsaved": true, "metadata": map[string]interface{}{"api_key": "private-fixture", "observacao": "Texto atual"}}}, "contexto antigo", &models.User{UsageMode: &mode})
	require.Contains(t, context, `"usage_mode":"explorer"`)
	require.Contains(t, context, `"usage_mode_label":"Explorador"`)
	require.Contains(t, context, `"living_clue_id":42`)
	require.Contains(t, context, "Rascunho")
	require.Contains(t, context, "Texto atual")
	for _, secret := range []string{"not-for-context", "private-fixture", "user_id", "contexto antigo"} {
		require.NotContains(t, context, secret)
	}
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(strings.SplitN(context, "\n", 2)[1]), &parsed))
}

func TestHistoryKeepsLatestUsefulTurnWithinBudgetWhenNewestMessageIsLarge(t *testing.T) {
	history := []models.ChatMessage{{Role: "user", Content: "qual era a pergunta?"}, {Role: "assistant", Content: strings.Repeat("á", 50000)}}
	contents := buildGeminiContents(history, "tela atual", "continuação")
	require.GreaterOrEqual(t, len(contents), 3)
	require.Contains(t, contents[0].Parts[0].Text, "qual era")
	require.Less(t, len([]rune(contents[1].Parts[0].Text)), 8100)
	require.Equal(t, "continuação", contents[len(contents)-1].Parts[0].Text)
}

func TestGeminiZeroClientReturnsSafeCancellationInsteadOfPanicking(t *testing.T) {
	t.Setenv("GENAI_API_KEY", "synthetic-test-key")
	t.Setenv("GEMINI_MODEL", "gemini-3.8-flash")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	reply, err := (&GeminiChatService{}).GenerateReply(ctx, ChatCompletionInput{Message: "synthetic cancellation"})
	require.Empty(t, reply)
	require.Error(t, err)
	require.Contains(t, err.Error(), "tempo")
	require.NotContains(t, err.Error(), "synthetic-test-key")
}
