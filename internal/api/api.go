package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/tusharsanka4/mini-miniredis/internal/commands"
	"github.com/tusharsanka4/mini-miniredis/internal/store"
)

const maxBodySize = 2 * 1024 * 1024

func Start(address string, db *store.Store) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer listener.Close()
	fmt.Println("MiniRedis HTTP API listening on", listener.Addr())
	s := &http.Server{
		Handler:           NewHandler(db),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return s.Serve(listener)
}

// NewHandler uses the supplied store, so HTTP and TCP can share the same data.
func NewHandler(db *store.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			if r.Method != http.MethodGet {
				w.Header().Set("Allow", "GET")
				writeError(w, http.StatusMethodNotAllowed, "use GET")
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		case "/command":
			if r.Method != http.MethodPost {
				w.Header().Set("Allow", "POST")
				writeError(w, http.StatusMethodNotAllowed, "use POST")
				return
			}
			executeCommand(w, r, db)
		default:
			writeError(w, http.StatusNotFound, "endpoint not found")
		}
	})
}

func executeCommand(w http.ResponseWriter, r *http.Request, db *store.Store) {
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}

	var request struct {
		Command []any `json:"command"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&request)
	if err == nil {
		// Reject a second JSON value or trailing garbage before executing a write.
		var extra any
		err = decoder.Decode(&extra)
		if err == io.EOF {
			err = nil
		} else if err == nil {
			err = errors.New("multiple JSON values")
		}
	}
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body exceeds 2 MiB")
		} else {
			writeError(w, http.StatusBadRequest, "invalid JSON request")
		}
		return
	}
	if len(request.Command) == 0 || len(request.Command) > 1024 {
		writeError(w, http.StatusBadRequest, "command must contain 1 to 1024 strings")
		return
	}
	args := make([]string, len(request.Command))
	for i, value := range request.Command {
		arg, ok := value.(string)
		if !ok {
			writeError(w, http.StatusBadRequest, "command arguments must be strings")
			return
		}
		if len(arg) > 1024*1024 {
			writeError(w, http.StatusBadRequest, "command arguments must not exceed 1 MiB")
			return
		}
		args[i] = arg
	}

	reply := commands.Dispatch(db, args)
	if strings.HasPrefix(reply, "-") {
		writeError(w, http.StatusBadRequest, strings.TrimSuffix(reply[1:], "\r\n"))
		return
	}
	result, err := decodeReply(reply)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not decode command reply")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": result})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}
