package tritonhttp

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
)

const (
	responseProto    = "HTTP/1.1"
	statusOK         = 200
	statusBadRequest = 400
	statusNotFound   = 404
)

var statusText = map[int]string{
	statusOK:         "OK",
	statusBadRequest: "Bad Request",
	statusNotFound:   "Not Found",
}

type Response struct {
	StatusCode int    // e.g. 200
	Proto      string // e.g. "HTTP/1.1"

	// Header stores all headers to write to the response.
	// Header keys are case-incensitive, and should be stored
	// in the canonical format in this map.
	Header map[string]string

	// Request is the valid request that leads to this response.
	// It could be nil for responses not resulting from a valid request.
	Request *Request

	// FilePath is the local path to the file to serve.
	// It could be "", which means there is no file to serve.
	FilePath string
}

// Write writes the res to the w.
func (res *Response) Write(w io.Writer) error {
	if err := res.WriteStatusLine(w); err != nil {
		return err
	}
	if err := res.WriteSortedHeaders(w); err != nil {
		return err
	}
	if err := res.WriteBody(w); err != nil {
		return err
	}
	return nil
}

// WriteStatusLine writes the status line of res to w, including the ending "\r\n".
// For example, it could write "HTTP/1.1 200 OK\r\n".
func (res *Response) WriteStatusLine(w io.Writer) error {
	bw := bufio.NewWriter(w)
	status_line := fmt.Sprintf("%v %v %v\r\n", responseProto, res.StatusCode, statusText[res.StatusCode])
	_, err := bw.WriteString(status_line)
	if err != nil {
		return err
	}

	err = bw.Flush()
	if err != nil {
		return err
	}
	return nil
}

// WriteSortedHeaders writes the headers of res to w, including the ending "\r\n".
// For example, it could write "Connection: close\r\nDate: foobar\r\n\r\n".
// For HTTP, there is no need to write headers in any particular order.
// TritonHTTP requires to write in sorted order for the ease of testing.
func (res *Response) WriteSortedHeaders(w io.Writer) error {
	keys := make([]string, 0, len(res.Header))
	for k := range res.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	bw := bufio.NewWriter(w)
	headers := ""
	for _, key := range keys {
		headers += fmt.Sprintf("%v: %v\r\n", key, res.Header[key])
	}
	headers += "\r\n"
	_, err := bw.WriteString(headers)
	if err != nil {
		return err
	}

	err = bw.Flush()
	if err != nil {
		return err
	}
	return nil
}

// WriteBody writes res' file content as the response body to w.
// It doesn't write anything if there is no file to serve.
func (res *Response) WriteBody(w io.Writer) error {
	if res.FilePath == "" {
		return nil
	} else {
		bw := bufio.NewWriter(w)
		body, err := os.ReadFile(res.FilePath)
		if err != nil {
			return err
		}
		_, err = bw.WriteString(string(body))
		if err != nil {
			return err
		}

		err = bw.Flush()
		if err != nil {
			return err
		}
		return nil
	}

}
