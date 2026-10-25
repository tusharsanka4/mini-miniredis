package resp

import "strconv"

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
