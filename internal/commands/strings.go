package commands

import (
	"github.com/tusharsanka4/mini-miniredis/internal/resp"
	"github.com/tusharsanka4/mini-miniredis/internal/store"
)

func set(db *store.Store, key, value string) string {
	db.Set(key, value)
	return resp.SimpleString("OK")
}

func get(db *store.Store, key string) string {
	value, exists := db.Get(key)
	if !exists {
		return resp.NullBulkString()
	}
	return resp.BulkString(value)
}

func del(db *store.Store, keys []string) string {
	return resp.Integer(db.Delete(keys...))
}
