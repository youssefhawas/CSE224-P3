package tritonhttp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	// Addr specifies the TCP address for the server to listen on,
	// in the form "host:port". It shall be passed to net.Listen()
	// during ListenAndServe().
	Addr string // e.g. ":0"

	// DocRoot specifies the path to the directory to serve static files from.
	DocRoot string
}

// ListenAndServe listens on the TCP network address s.Addr and then
// handles requests on incoming connections.
func (s *Server) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		log.Panic(err)
	}

	fmt.Println("Listening on ", listener.Addr().String())

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Panic(err)
		}
		go s.HandleConnection(conn)
	}
	// Hint: call HandleConnection
}

// HandleConnection reads requests from the accepted conn and handles them.
func (s *Server) HandleConnection(conn net.Conn) {
	// Hint: use the other methods below
	br := bufio.NewReader(conn)
	for {
		// Set timeout
		err := conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		if err != nil {
			log.Printf("Failed to set timeout for connection %v", conn)
			conn.Close()
			return
		}
		req, bytesReceived, err := ReadRequest(br)

		//Handle EOF
		if errors.Is(err, io.EOF) {
			log.Printf("Connection closed by %v", conn.RemoteAddr())
			_ = conn.Close()
			return
		}

		// Handle timeout
		if err, ok := err.(net.Error); ok && err.Timeout() {
			fmt.Println(bytesReceived)
			if bytesReceived {
				// Send 400 response
				res := &Response{}
				res.HandleBadRequest()
				err := res.Write(conn)
				if err != nil {
					fmt.Println(err)
				}
				conn.Close()
				return
			}
			_ = conn.Close()
			return
		}

		// Handle bad request
		if err != nil {
			fmt.Println(err)
			res := &Response{}
			res.HandleBadRequest()
			err := res.Write(conn)
			if err != nil {
				fmt.Println(err)
			}
			conn.Close()
			return
		}

		// Handle good request
		if err == nil {
			res := s.HandleGoodRequest(req)
			err := res.Write(conn)
			if err != nil {
				fmt.Println(err)
			}
			if res.StatusCode == 404 {
				conn.Close()
				return
			}
		}

		// Close conn if requested
		if req.Close {
			fmt.Println("closing connection (connection header)")
			conn.Close()
			return
		}

	}
}

// HandleGoodRequest handles the valid req and generates the corresponding res.
func (s *Server) HandleGoodRequest(req *Request) (res *Response) {
	// Hint: use the other methods below
	res = &Response{}
	url := req.URL
	full_path := filepath.Join(s.DocRoot, url)
	cleaned_path := filepath.Clean(full_path)
	if !strings.Contains(cleaned_path, s.DocRoot) {
		res.HandleNotFound(req)
		return res
	}
	dir, err := os.Stat(cleaned_path)
	if err != nil {
		fmt.Println(cleaned_path)
		res.HandleNotFound(req)
		return res
	}
	if dir.IsDir() {
		cleaned_path = filepath.Join(cleaned_path, "index.html")
	}
	fmt.Println(cleaned_path)
	_, err = os.Stat(cleaned_path)
	if err != nil {
		res.HandleNotFound(req)
		return res
	}
	res.HandleOK(req, cleaned_path)
	return res
}

// HandleOK prepares res to be a 200 OK response
// ready to be written back to client.
func (res *Response) HandleOK(req *Request, path string) {
	headers := make(map[string]string)
	headers["Date"] = FormatTime(time.Now())
	res.FilePath = path

	fi, err := os.Stat(path)
	if err != nil {
		fmt.Println(err)
	}
	size := fi.Size()
	headers["Last-Modified"] = FormatTime(fi.ModTime())
	headers["Content-Length"] = strconv.Itoa(int(size))
	headers["Content-Type"] = MIMETypeByExtension(filepath.Ext(path))
	if req.Close {
		headers["Connection"] = "close"
	}
	res.StatusCode = 200
	res.Header = headers
	res.Proto = "HTTP/1.1"
	res.Request = req
}

// HandleBadRequest prepares res to be a 400 Bad Request response
// ready to be written back to client.
func (res *Response) HandleBadRequest() {
	headers := make(map[string]string)
	headers["Date"] = FormatTime(time.Now())
	headers["Connection"] = "close"
	res.StatusCode = 400
	res.Header = headers
	res.Proto = "HTTP/1.1"
	fmt.Println(res)
}

// HandleNotFound prepares res to be a 404 Not Found response
// ready to be written back to client.
func (res *Response) HandleNotFound(req *Request) {
	headers := make(map[string]string)
	headers["Date"] = FormatTime(time.Now())
	if req.Close {
		headers["Connection"] = "close"
	}
	res.StatusCode = 404
	res.Header = headers
	res.Proto = "HTTP/1.1"
}
