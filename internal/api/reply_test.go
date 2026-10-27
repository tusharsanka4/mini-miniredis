package api

import (
	"reflect"
	"strings"
	"testing"

	"github.com/tusharsanka4/mini-miniredis/internal/resp"
)

func TestDecodeReply(t *testing.T) {
	large := strings.Repeat("x", 5000)
	for _, test := range []struct {
		reply string
		want  any
	}{
		{"+OK\r\n", "OK"},
		{":3\r\n", int64(3)},
		{"$-1\r\n", nil},
		{"$0\r\n\r\n", ""},
		{"$4\r\na\r\nb\r\n", "a\r\nb"},
		{resp.BulkString(large), large},
		{"*0\r\n", []any{}},
		{"*2\r\n$1\r\na\r\n$1\r\nb\r\n", []any{"a", "b"}},
	} {
		got, err := decodeReply(test.reply)
		if err != nil || !reflect.DeepEqual(got, test.want) {
			t.Errorf("reply: got %v, %v; want %v", got, err, test.want)
		}
	}
	// Response arrays can exceed the request parser's 1024-argument limit.
	members := make([]string, 1025)
	got, err := decodeReply(resp.Array(members))
	if err != nil || len(got.([]any)) != len(members) {
		t.Fatal("large response array was not decoded", err)
	}
	for _, reply := range []string{"", "+OK\n", ":x\r\n", "$-2\r\n", "$2\r\na", "$1\r\naxx", "*1\r\n", "?bad\r\n"} {
		if _, err := decodeReply(reply); err == nil {
			t.Errorf("accepted invalid reply %q", reply)
		}
	}
}
