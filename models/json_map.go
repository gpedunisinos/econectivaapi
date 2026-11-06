package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSONMap persists a generic map as a JSON column.
type JSONMap map[string]interface{}

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	b, err := json.Marshal(j)
	return string(b), err
}

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	var bytesValue []byte
	switch v := value.(type) {
	case []byte:
		bytesValue = v
	case string:
		bytesValue = []byte(v)
	default:
		return errors.New("unsupported type for JSONMap")
	}
	return json.Unmarshal(bytesValue, j)
}
