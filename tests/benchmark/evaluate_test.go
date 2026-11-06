package benchmark

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/stretchr/testify/require"
)

func TestRequiredMatrixAndSelection(t *testing.T) {
	cases, err := LoadCases()
	require.NoError(t, err)
	require.Len(t, cases, 22)
	chosen, err := SelectCases(cases, []string{"pista-chat-fonte-externa"}, nil)
	require.NoError(t, err)
	require.Len(t, chosen, 1)
	chosen, err = SelectCases(cases, nil, []string{"seguranca"})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(chosen), 4)
	_, err = SelectCases(cases, []string{"does-not-exist"}, nil)
	require.Error(t, err)
	_, err = SelectCases(cases, nil, []string{"does-not-exist"})
	require.Error(t, err)
}
func TestEvaluatorDoesNotApproveKeywordOnlyWrongIDsMissingDataOrProviderFailures(t *testing.T) {
	step := Step{Expectations: Expectations{RequiredPatterns: []string{"18 árvores", "viveiro-escola"}, ForbiddenPatterns: []string{"27 árvores", "não possui 18"}, Calls: []CallExpectation{{Name: "pista_obter", Arguments: map[string]interface{}{"id": ClueA}, ResultPatterns: []string{"18 árvores", "viveiro-escola"}}}}}
	trace := services.ChatTrace{Calls: []services.ToolExecution{{Name: "pista_obter", Arguments: json.RawMessage(`{"id":910101}`), Result: map[string]interface{}{"description": "18 árvores", "origem": "viveiro-escola"}}}}
	good := "A Horta Aurora possui 18 árvores e origem viveiro-escola."
	require.Empty(t, Evaluate(step, good, trace, nil))
	for _, bad := range []string{"18 árvores", good + " Na verdade são 27 árvores.", "A Horta não possui 18 árvores; viveiro-escola."} {
		require.NotEmpty(t, Evaluate(step, bad, trace, nil))
	}
	wrong := trace
	wrong.Calls = append([]services.ToolExecution{}, trace.Calls...)
	wrong.Calls[0].Arguments = json.RawMessage(`{"id":910102}`)
	require.NotEmpty(t, Evaluate(step, good, wrong, nil))
	absent := trace
	absent.Calls = append([]services.ToolExecution{}, trace.Calls...)
	absent.Calls[0].Result = map[string]interface{}{"name": "Horta Aurora"}
	require.NotEmpty(t, Evaluate(step, good, absent, nil))
	for _, err := range []error{services.ErrGeminiQuota, services.ErrGeminiUnavailable, services.ErrGeminiModel} {
		require.NotEmpty(t, Evaluate(step, good, trace, err))
	}
	require.NotEmpty(t, Evaluate(step, "", trace, nil))
}
func TestOnlyConfiguredOrchestratorLimitMayBeExpectedNotProviderErrors(t *testing.T) {
	step := Step{Expectations: Expectations{ExpectedLimitError: true, RequiredPatterns: []string{"limite de consultas"}}}
	require.Empty(t, Evaluate(step, "", services.ChatTrace{}, services.ErrQueryLimit))
	require.NotEmpty(t, Evaluate(step, "", services.ChatTrace{}, services.ErrGeminiQuota))
}

func TestHistoryDenialAcceptsValidParaphrasesAndRejectsARealLeak(t *testing.T) {
	cases, err := LoadCases()
	require.NoError(t, err)
	var probe Step
	for _, c := range cases {
		if c.ID == "historicos-separados" {
			probe = c.Steps[1]
		}
	}
	require.NotEmpty(t, probe.Message)
	for _, reply := range []string{
		"No histórico da nossa conversa atual não há registro de mensagens anteriores nem de uma palavra de teste que você tenha dito.",
		"Não há registro de mensagens anteriores nesta nossa conversa, por isso não tenho como saber qual palavra de teste você mencionou antes.",
		"Não tenho registro de mensagens anteriores nesta conversa nem de uma palavra de teste informada antes.",
	} {
		require.Empty(t, Evaluate(probe, reply, services.ChatTrace{}, nil))
	}
	require.NotEmpty(t, Evaluate(probe, "Não tenho registro anterior, mas a palavra é ORQUIDEA-AZUL.", services.ChatTrace{}, nil))
}

func TestUSPParaphrasesKeepFactualRequirements(t *testing.T) {
	cases, err := LoadCases()
	require.NoError(t, err)
	patterns := map[string][]string{}
	for _, c := range cases {
		patterns[c.ID] = c.Steps[0].Expectations.RequiredPatterns
	}
	require.Regexp(t, "(?is)"+patterns["usp-contexto-oito-indicadores"][2], "Cobertura vegetal: 0%")
	require.NotRegexp(t, "(?is)"+patterns["usp-contexto-oito-indicadores"][2], "Cobertura vegetal: não informada")
	require.Regexp(t, patterns["usp-leitura-antiga-sem-diagnostico"][2], "01/09/2026 às 10:00")
	require.NotRegexp(t, patterns["usp-leitura-antiga-sem-diagnostico"][2], "02/09/2026")
	stale, err := regexp.Compile("(?is)" + patterns["usp-leitura-antiga-sem-diagnostico"][3])
	require.NoError(t, err)
	require.True(t, stale.MatchString("São dados atuais?\n* Não. Os registros são de setembro."))
	require.False(t, stale.MatchString("São dados atuais? Sim, em tempo real."))
	for _, c := range cases {
		if c.ID != "iarete-edicao-rascunho" {
			continue
		}
		for _, pattern := range c.Steps[0].Expectations.ForbiddenPatterns {
			forbidden, err := regexp.Compile("(?is)" + pattern)
			require.NoError(t, err)
			require.False(t, forbidden.MatchString("Para que a alteração seja confirmada, clique Salvar."))
		}
	}
}
