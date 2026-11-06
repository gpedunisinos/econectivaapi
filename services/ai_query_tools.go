package services

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/OdairPianta/unisinos_usp_api/models"
)

// QuerySession is implemented by the official MCP client. Keeping this contract
// here avoids a services/repository dependency cycle and permits isolated tests.
type QuerySession interface {
	Specifications() []QueryToolSpec
	Call(context.Context, string, json.RawMessage) (map[string]interface{}, error)
	Close() error
}

type QueryToolSpec struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type QuerySessionFactory func(context.Context, models.User, uint) (QuerySession, error)

var queryFactory struct {
	sync.RWMutex
	new QuerySessionFactory
}

func SetQuerySessionFactory(factory QuerySessionFactory) {
	queryFactory.Lock()
	queryFactory.new = factory
	queryFactory.Unlock()
}

func GetQuerySessionFactory() QuerySessionFactory {
	queryFactory.RLock()
	defer queryFactory.RUnlock()
	return queryFactory.new
}
