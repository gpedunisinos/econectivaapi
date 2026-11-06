package requests

import (
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator/v10"
	"testing"
)

func TestSampleDoubleDistinguishesZeroFromMissing(t *testing.T) {
	v := validator.New()
	v.SetTagName("binding")
	// Foreign-key and uniqueness validation belong to HTTP/database tests.
	for _, tag := range []string{"exists", "not_exists"} {
		if err := v.RegisterValidation(tag, func(validator.FieldLevel) bool { return true }); err != nil {
			t.Fatal(err)
		}
	}
	for _, target := range []string{"create", "update"} {
		for _, tc := range []struct {
			name, field string
			valid       bool
		}{
			{"zero", `,"sample_double":0`, true}, {"nonzero", `,"sample_double":12.5`, true},
			{"missing", ``, false}, {"null", `,"sample_double":null`, false},
		} {
			t.Run(target+"/"+tc.name, func(t *testing.T) {
				body := fmt.Sprintf(`{"name":"Synthetic","sample_string":"Text","sample_unique":"unique","sample_date":"2026-10-26T00:00:00Z","sample_detail_id":2,"order_number":1%s}`, tc.field)
				var input interface{}
				if target == "create" {
					input = &CreateSampleModelInput{}
				} else {
					input = &UpdateSampleModelInput{}
				}
				if err := json.Unmarshal([]byte(body), input); err != nil {
					t.Fatal(err)
				}
				err := v.Struct(input)
				if (err == nil) != tc.valid {
					t.Fatalf("valid=%v, validation=%v", tc.valid, err)
				}
			})
		}
	}
}
