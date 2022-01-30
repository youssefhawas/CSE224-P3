package tritonhttp

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Request struct {
	Method string // e.g. "GET"
	URL    string // e.g. "/path/to/a/file"
	Proto  string // e.g. "HTTP/1.1"

	// Header stores misc headers excluding "Host" and "Connection",
	// which are stored in special fields below.
	// Header keys are case-incensitive, and should be stored
	// in the canonical format in this map.
	Header map[string]string

	Host  string // determine from the "Host" header
	Close bool   // determine from the "Connection" header
}

// ReadRequest tries to read the next valid request from br.
//
// If it succeeds, it returns the valid request read. In this case,
// bytesReceived should be true, and err should be nil.
//
// If an error occurs during the reading, it returns the error,
// and a nil request. In this case, bytesReceived indicates whether or not
// some bytes are received before the error occurs. This is useful to determine
// the timeout with partial request received condition.
func ReadRequest(br *bufio.Reader) (req *Request, bytesReceived bool, err error) {
	req = &Request{}
	bytesReceived = false
	initial_line, err := ReadLine(br)
	if err != nil {
		return nil, bytesReceived, err
	}

	req.Method, req.URL, req.Proto, err = parseInitialLine(initial_line)
	if err != nil {
		return nil, bytesReceived, err
	}
	bytesReceived = true

	req.Header = make(map[string]string)
	// Read headers
	for {
		header_line, err := ReadLine(br)
		if err != nil {
			if err == io.EOF {
				continue
			} else {
				return nil, bytesReceived, err
			}
		}
		if header_line == "" {
			break
		}
		split_line := strings.SplitN(header_line, ":", 2)
		key := CanonicalHeaderKey(split_line[0])
		val := split_line[1]
		if val[0] == ' ' {
			val = strings.ReplaceAll(val, " ", "")
		}
		req.Header[key] = val
	}

	// Check required headers
	val, ok := req.Header["Host"]
	if !ok {
		return nil, bytesReceived, fmt.Errorf("missing required header")
	} else {
		req.Host = val
		delete(req.Header, "Host")
	}

	// Handle special headers
	val, ok = req.Header["Connection"]
	if ok {
		if val != "close" {
			req.Close = false
		} else {
			req.Close = true
		}
		delete(req.Header, "Connection")
	}
	fmt.Printf("REQUEST %v", req)
	return req, bytesReceived, nil
}

func parseInitialLine(initial_line string) (string, string, string, error) {
	split_line := strings.Split(initial_line, " ")
	if len(split_line) != 3 {
		return "", "", "", fmt.Errorf("could not parse the request line, got fields %v", split_line)
	}
	method := split_line[0]
	url := split_line[1]
	proto := split_line[2]

	if method != "GET" {
		return method, url, proto, fmt.Errorf("invalid method received, got method %v", method)
	}

	if proto != "HTTP/1.1" {
		return method, url, proto, fmt.Errorf("invalid proto received, got proto %v", proto)
	}

	return method, url, proto, nil
}
