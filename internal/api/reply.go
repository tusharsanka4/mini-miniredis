package api

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// The existing dispatcher returns RESP. Decode its replies here so HTTP callers
// receive ordinary JSON values without needing to understand the TCP protocol.
func decodeReply(reply string) (any, error) {
	return readReply(bufio.NewReader(strings.NewReader(reply)))
}

func readReply(reader *bufio.Reader) (any, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if len(line) < 3 || !strings.HasSuffix(line, "\r\n") {
		return nil, fmt.Errorf("invalid reply header")
	}
	value := line[1 : len(line)-2]
	switch line[0] {
	case '+':
		return value, nil
	case ':':
		return strconv.ParseInt(value, 10, 64)
	case '$', '*':
		length, err := strconv.Atoi(value)
		if err != nil || length < -1 {
			return nil, fmt.Errorf("invalid reply length")
		}
		if length == -1 {
			return nil, nil
		}
		if line[0] == '$' {
			data := make([]byte, length+2)
			if _, err := io.ReadFull(reader, data); err != nil {
				return nil, err
			}
			if string(data[length:]) != "\r\n" {
				return nil, fmt.Errorf("invalid string terminator")
			}
			return string(data[:length]), nil
		}
		values := make([]any, length)
		for i := range values {
			values[i], err = readReply(reader)
			if err != nil {
				return nil, err
			}
		}
		return values, nil
	default:
		return nil, fmt.Errorf("unsupported reply type")
	}
}
