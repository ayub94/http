package server

import(
	"net"
	"sync"
	"log"
//	"os"
	"io"
	"bytes"
	"strings"
	"net/url"


)

type HandlerFunc func(conn net.Conn)

type Server struct{
	addr     string
	mu       sync.RWMutex
	handlers   map[string]HandlerFunc
}
type Request struct {
	Conn        net.Conn
	QueryParams url.Values
	PathParams  map[string]string
	Headers     map[string]string
	Body        []byte
}

func NewServer(addr string) *Server {
	return &Server{addr: addr, handlers: make(map[string]HandlerFunc)}
}

func(s *Server) Register(path string, handler HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[path] = handler
}

func (s *Server) Start() (err error) {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.Println(err)
		return err
	}
	defer func() {
		if cerr := listener.Close(); cerr != nil {
			err = cerr
			return
		}
	}()
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		go s.handle(conn)

	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
  
	buf := make([]byte, 4096)
	for {
	  n, err := conn.Read(buf)
	  if err == io.EOF {
		log.Printf("%s", buf[:n])
	  }
	  if err != nil {
		log.Println(err)
		return
	  }
	  var req Request
  
	  data := buf[:n]
	  requestLineDelim := []byte{'\r', '\n'}
	  requestLineEnd := bytes.Index(data,requestLineDelim)
	  if requestLineEnd == -1 {
		log.Println("ErrBadRequest")
		return
	  }

	  headersLineDelim:=[]byte{'\r', '\n', '\r', '\n'}
	  headersLineEnd := bytes.Index(data, headersLineDelim)
	  if requestLineEnd == -1 {
		return
	  }

	  headersLine := string(data[requestLineEnd:headersLineEnd])
	  headers := strings.Split(headersLine, "\r\n")[1:]

	  mp := make(map[string]string)
	  for _, v := range headers {
		headerLine := strings.Split(v, ": ")
		mp[headerLine[0]] = headerLine[1]
	  }

	  req.Headers = mp

	  req.Body=data[headersLineEnd+4:]
  
	  reqLine := string(data[:requestLineEnd])
	  parts := strings.Split(reqLine, " ")
  
	  if len(parts) != 3 {
		log.Println("ErrBadRequest")
		return
	  }
  
	  path, version := parts[1], parts[2]
  
	  if version != "HTTP/1.1" {
		log.Println("ErrHTTPVersionNotValid")
		return
	  }

	  decoded, err:=url.PathUnescape(path)
	  if err!=nil {
		  log.Println(err)
		  return
	  }
	  log.Println(decoded)

	  uri, err:=url.ParseRequestURI(decoded)
	  if err!=nil {
		log.Println(err)
		return
	  }
	  
	  req.Conn = conn
	  req.QueryParams = uri.Query()
  
	  var handler = func(conn net.Conn) {
		conn.Close()
	  }
	  s.mu.RLock()
	  for i := 0; i < len(s.handlers); i++ {
		if hdlr, found := s.handlers[path]; found {
		  handler = hdlr
		  break
		}
	  }
	  s.mu.RUnlock()
	  handler(conn) 
	} 
}