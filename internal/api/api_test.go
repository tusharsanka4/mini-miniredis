package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tusharsanka4/mini-miniredis/internal/store"
)

func TestCommands(t *testing.T) {
	db := store.New()
	handler := NewHandler(db)
	cases := []struct {
		body string
		want string
	}{
		{`{"command":["PING"]}`, `{"result":"PONG"}`},
		{`{"command":["GET","missing"]}`, `{"result":null}`},
		{`{"command":["SET","name","tushar"]}`, `{"result":"OK"}`},
		{`{"command":["GET","name"]}`, `{"result":"tushar"}`},
		{`{"command":["SET","empty",""]}`, `{"result":"OK"}`},
		{`{"command":["GET","empty"]}`, `{"result":""}`},
		{`{"command":["ECHO","hello\r\n世界"]}`, `{"result":"hello\r\n世界"}`},
		{`{"command":["LRANGE","missing","0","-1"]}`, `{"result":[]}`},
		{`{"command":["RPUSH","tasks","first","second"]}`, `{"result":2}`},
		{`{"command":["LRANGE","tasks","0","-1"]}`, `{"result":["first","second"]}`},
		{`{"command":["SADD","tags","go","go"]}`, `{"result":1}`},
		{`{"command":["SMEMBERS","tags"]}`, `{"result":["go"]}`},
		{`{"command":["HSET","user","name","tushar"]}`, `{"result":1}`},
		{`{"command":["HGETALL","user"]}`, `{"result":["name","tushar"]}`},
		{`{"command":["ZADD","scores","2","bob","1","alice"]}`, `{"result":2}`},
		{`{"command":["ZRANGE","scores","0","-1","WITHSCORES"]}`, `{"result":["alice","1","bob","2"]}`},
		{`{"command":["DEL","name"]}`, `{"result":1}`},
	}
	for _, test := range cases {
		r := httptest.NewRequest(http.MethodPost, "/command", strings.NewReader(test.body))
		r.Header.Set("Content-Type", "application/json; charset=utf-8")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != test.want {
			t.Fatalf("%s: status %d, reply %s, want %s", test.body, w.Code, w.Body, test.want)
		}
		if w.Header().Get("Content-Type") != "application/json" {
			t.Fatal("missing JSON content type")
		}
	}
	// The handler uses the supplied store, not a separate HTTP-only copy.
	if db.Type("tasks") != "list" {
		t.Fatal("HTTP write was not visible through the supplied store")
	}
	db.Set("direct", "visible")
	r := httptest.NewRequest(http.MethodPost, "/command", strings.NewReader(`{"command":["GET","direct"]}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if strings.TrimSpace(w.Body.String()) != `{"result":"visible"}` {
		t.Fatalf("store write not visible over HTTP: %s", w.Body)
	}
}

func TestRequestErrorsDoNotWrite(t *testing.T) {
	cases := []struct {
		name        string
		body        string
		contentType string
		status      int
	}{
		{"empty body", "", "application/json", 400},
		{"malformed", `{`, "application/json", 400},
		{"missing command", `{}`, "application/json", 400},
		{"null body", `null`, "application/json", 400},
		{"empty command", `{"command":[]}`, "application/json", 400},
		{"null command", `{"command":null}`, "application/json", 400},
		{"not an array", `{"command":"PING"}`, "application/json", 400},
		{"number argument", `{"command":["SET","key",3]}`, "application/json", 400},
		{"null argument", `{"command":["SET","key",null]}`, "application/json", 400},
		{"unknown field", `{"command":["SET","key","value"],"extra":true}`, "application/json", 400},
		{"trailing object", `{"command":["SET","key","value"]} {}`, "application/json", 400},
		{"trailing garbage", `{"command":["SET","key","value"]} x`, "application/json", 400},
		{"missing content type", `{"command":["SET","key","value"]}`, "", 415},
		{"wrong content type", `{"command":["SET","key","value"]}`, "text/plain", 415},
		{"oversized body", `{"command":["SET","key","` + strings.Repeat("x", maxBodySize) + `"]}`, "application/json", 413},
		{"oversized argument", `{"command":["SET","key","` + strings.Repeat("x", 1024*1024+1) + `"]}`, "application/json", 400},
		{"too many arguments", `{"command":["DEL"` + strings.Repeat(`,"key"`, 1024) + `]}`, "application/json", 400},
		{"wrong arity", `{"command":["SET","key"]}`, "application/json", 400},
		{"unknown command", `{"command":["NOPE"]}`, "application/json", 400},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			db := store.New()
			r := httptest.NewRequest(http.MethodPost, "/command", strings.NewReader(test.body))
			r.Header.Set("Content-Type", test.contentType)
			w := httptest.NewRecorder()
			NewHandler(db).ServeHTTP(w, r)
			if w.Code != test.status {
				t.Fatalf("status %d, want %d: %s", w.Code, test.status, w.Body)
			}
			var response map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || response["error"] == "" {
				t.Fatalf("expected JSON error: %s", w.Body)
			}
			if db.Type("key") != "none" {
				t.Fatal("invalid request changed the store")
			}
		})
	}
}

func TestRoutesAndWrongType(t *testing.T) {
	db := store.New()
	db.Set("name", "tushar")
	handler := NewHandler(db)
	for _, test := range []struct {
		method, path, body string
		status             int
		want               string
		allow              string
	}{
		{"GET", "/health", "", 200, `{"status":"ok"}`, ""},
		{"POST", "/health", "", 405, `{"error":"use GET"}`, "GET"},
		{"GET", "/command", "", 405, `{"error":"use POST"}`, "POST"},
		{"GET", "/unknown", "", 404, `{"error":"endpoint not found"}`, ""},
		{"POST", "/command", `{"command":["LPUSH","name","item"]}`, 400, `{"error":"WRONGTYPE Operation against a key holding the wrong kind of value"}`, ""},
	} {
		r := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != test.status || strings.TrimSpace(w.Body.String()) != test.want || w.Header().Get("Allow") != test.allow {
			t.Errorf("%s %s: status %d, body %s, Allow %q", test.method, test.path, w.Code, w.Body, w.Header().Get("Allow"))
		}
	}
}
