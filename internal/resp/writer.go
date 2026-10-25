package resp

import (
	"strconv"
	"strings"
)

func SimpleString(value string) string {
	return "+" + value + "\r\n"
}

func BulkString(value string) string {
	return "$" + strconv.Itoa(len(value)) + "\r\n" + value + "\r\n"
}

func NullBulkString() string {
	return "$-1\r\n"
}

func Error(message string) string {
	return "-ERR " + message + "\r\n"
}

func Integer(value int) string {
	return ":" + strconv.Itoa(value) + "\r\n"
}

func WrongType() string {
	return "-WRONGTYPE Operation against a key holding the wrong kind of value\r\n"
}

func Array(values []string) string {
	var reply strings.Builder
	reply.WriteString("*" + strconv.Itoa(len(values)) + "\r\n")
	for _, value := range values {
		reply.WriteString(BulkString(value))
	}
	return reply.String()
}
