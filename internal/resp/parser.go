package resp

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
)

// Commands are RESP arrays of bulk strings, e.g. *2\r\n$3\r\nGET\r\n$3\r\nkey\r\n.
func ReadCommand(reader *bufio.Reader) ([]string, error) {
	count, err := readLength(reader, '*')
	if err != nil {
		return nil, err
	}
	if count < 1 || count > 1024 {
		return nil, fmt.Errorf("invalid argument count")
	}

	args := make([]string, count)
	for i := range args {
		length, err := readLength(reader, '$')
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return nil, err
		}
		// Keep a malformed client from requesting an enormous allocation.
		if length < 0 || length > 1024*1024 {
			return nil, fmt.Errorf("invalid bulk string length")
		}
		data := make([]byte, length+2)
		if _, err := io.ReadFull(reader, data); err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return nil, err
		}
		if string(data[length:]) != "\r\n" {
			return nil, fmt.Errorf("missing bulk string terminator")
		}
		args[i] = string(data[:length])
	}
	return args, nil
}

func readLength(reader *bufio.Reader, prefix byte) (int, error) {
	// ReadSlice also rejects headers that exceed the reader's buffer.
	line, err := reader.ReadSlice('\n')
	if err != nil {
		if err == io.EOF && len(line) > 0 {
			err = io.ErrUnexpectedEOF
		}
		return 0, err
	}
	if len(line) < 4 || line[0] != prefix || string(line[len(line)-2:]) != "\r\n" {
		return 0, fmt.Errorf("invalid length header")
	}
	return strconv.Atoi(string(line[1 : len(line)-2]))
}
