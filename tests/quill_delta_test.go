package tests

import (
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/services"
)

func TestValidateQuillDelta(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "empty", value: "", wantErr: false},
		{name: "valid", value: `[{"insert":"Texto formatado"},{"insert":"\n","attributes":{"header":1}}]`, wantErr: false},
		{name: "invalid json", value: "plain text", wantErr: true},
		{name: "object instead of array", value: `{"insert":"texto"}`, wantErr: true},
		{name: "operation without insert", value: `[{"retain":1}]`, wantErr: true},
		{name: "opaque legacy image is preserved", value: `[{"insert":{"image":"https://example.com/image.png"}}]`, wantErr: false},
		{name: "active video still rejects an unsupported origin", value: `[{"insert":{"video":"https://example.com/video.mp4"}}]`, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := services.ValidateQuillDelta(test.value)
			if (err != nil) != test.wantErr {
				t.Fatalf("ValidateQuillDelta() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
