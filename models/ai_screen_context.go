package models

// AIScreenContext is UI data, never an authorization token. IDs are reloaded
// and checked against the authenticated user before any database consultation.
type AIScreenContext struct {
	UsageModeLabel string                 `json:"usage_mode_label"`
	Version        int                    `json:"version"`
	ScreenName     string                 `json:"screen_name"`
	Route          string                 `json:"route"`
	Stage          string                 `json:"stage"`
	Goal           string                 `json:"goal"`
	UsageMode      string                 `json:"usage_mode"`
	References     map[string]uint        `json:"references,omitempty"`
	VisibleFields  []string               `json:"visible_fields,omitempty"`
	FilledFields   map[string]string      `json:"filled_fields,omitempty"`
	VisibleRecords []AIRecordReference    `json:"visible_records,omitempty"`
	Draft          map[string]interface{} `json:"draft,omitempty"`
	Notes          []string               `json:"notes,omitempty"`
}

type AIRecordReference struct {
	Kind string `json:"kind"`
	ID   uint   `json:"id"`
	Name string `json:"name"`
}
