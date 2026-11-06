package services

import "testing"

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
