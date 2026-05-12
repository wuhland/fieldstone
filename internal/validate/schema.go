package validate

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/xeipuuv/gojsonschema"
)

type entry struct {
	schema    *gojsonschema.Schema
	expiresAt time.Time
}

// Cache compiles and caches JSON Schemas with a 60-second TTL.
type Cache struct {
	mu    sync.RWMutex
	items map[string]*entry
	ttl   time.Duration
}

func NewCache() *Cache {
	return &Cache{
		items: make(map[string]*entry),
		ttl:   60 * time.Second,
	}
}

// Validate validates data against schemaJSON, using schemaKey as cache key.
// Returns validation error strings on failure, nil slice on success.
func (c *Cache) Validate(schemaKey string, schemaJSON json.RawMessage, data json.RawMessage) ([]string, error) {
	schema, err := c.compiled(schemaKey, schemaJSON)
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}

	result, err := schema.Validate(gojsonschema.NewBytesLoader(data))
	if err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	if result.Valid() {
		return nil, nil
	}

	errs := make([]string, 0, len(result.Errors()))
	for _, e := range result.Errors() {
		errs = append(errs, e.String())
	}
	return errs, nil
}

func (c *Cache) compiled(key string, schemaJSON json.RawMessage) (*gojsonschema.Schema, error) {
	c.mu.RLock()
	if e, ok := c.items[key]; ok && time.Now().Before(e.expiresAt) {
		s := e.schema
		c.mu.RUnlock()
		return s, nil
	}
	c.mu.RUnlock()

	compiled, err := gojsonschema.NewSchema(gojsonschema.NewBytesLoader(schemaJSON))
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.items[key] = &entry{schema: compiled, expiresAt: time.Now().Add(c.ttl)}
	c.mu.Unlock()

	return compiled, nil
}
