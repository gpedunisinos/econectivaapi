package benchmark

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/OdairPianta/unisinos_usp_api/services"
)

// Evaluate requires every configured assertion. Mentioning one expected word
// never compensates for a wrong ID, missing source, forbidden fact or error.
func Evaluate(step Step, response string, trace services.ChatTrace, err error) []string {
	failures := []string{}
	expected := step.Expectations
	text := response
	if expected.ExpectedLimitError && errors.Is(err, services.ErrQueryLimit) {
		text = err.Error()
	}
	if expected.NoCalls && len(trace.Calls) > 0 {
		failures = append(failures, "consultas desnecessárias para o contexto da interface")
	}
	if err != nil && !(expected.ExpectedLimitError && errors.Is(err, services.ErrQueryLimit)) {
		failures = append(failures, "execução falhou: "+err.Error())
	}
	if strings.TrimSpace(response) == "" && !(expected.ExpectedLimitError && errors.Is(err, services.ErrQueryLimit)) {
		failures = append(failures, "resposta vazia/inválida")
	}
	for _, pattern := range expected.RequiredPatterns {
		compiled, compileErr := regexp.Compile("(?is)" + pattern)
		if compileErr != nil {
			failures = append(failures, "asserção inválida: "+pattern)
			continue
		}
		if !compiled.MatchString(text) {
			failures = append(failures, "fato/origem obrigatória ausente: "+pattern)
		}
	}
	for _, pattern := range expected.ForbiddenPatterns {
		compiled, compileErr := regexp.Compile("(?is)" + pattern)
		if compileErr != nil {
			failures = append(failures, "proibição inválida: "+pattern)
			continue
		}
		if compiled.MatchString(text) {
			failures = append(failures, "comportamento/fato proibido: "+pattern)
		}
	}
	for _, want := range expected.Calls {
		found := false
		for _, call := range trace.Calls {
			if call.Name == want.Name && sameArguments(call.Arguments, want.Arguments) && (call.Error != "") == want.MustFail {
				found = true
				raw, _ := json.Marshal(call.Result)
				for _, pattern := range want.ResultPatterns {
					re, compileErr := regexp.Compile("(?is)" + pattern)
					if compileErr != nil || !re.Match(raw) {
						failures = append(failures, "resultado da consulta não contém o dado obrigatório: "+want.Name+" / "+pattern)
					}
				}
				break
			}
		}
		if !found {
			failures = append(failures, fmt.Sprintf("consulta/argumentos/resultado ausentes: %s %v (erro=%v)", want.Name, want.Arguments, want.MustFail))
		}
	}
	for _, call := range trace.Calls {
		for _, forbidden := range expected.ForbiddenCalls {
			if call.Name == forbidden {
				failures = append(failures, "consulta proibida: "+call.Name)
			}
		}
	}
	limit := expected.MaximumCalls
	if limit == 0 {
		limit = 8
	}
	if len(trace.Calls) > limit {
		failures = append(failures, "limite de consultas excedido")
	}
	return failures
}
