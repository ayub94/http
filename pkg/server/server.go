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

type HandlerFunc func(req *Request)

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

	  req.Body=data[headersLineEnd +4:]
  
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

  
	  var handler = func(req *Request) {
		req.Conn.Close()
	  }
	  s.mu.RLock()

	  pathParam, hr:=s.checkPath(uri.Path)
	  if hr!=nil {
		req.PathParams = pathParam
		handler = hr
	  }
	  s.mu.RUnlock()
	  handler(&req) 
	} 
}

func (s *Server) checkPath(path string) (map[string]string, HandlerFunc) {

	strRoutes := make([]string, len(s.handlers))
	i := 0
	for k := range s.handlers {
	  strRoutes[i] = k
	  i++
	}
  
	mp := make(map[string]string)
    
	for i := 0; i < len(strRoutes); i++ {
	  flag := false
	  route := strRoutes[i]
	  partsRoute := strings.Split(route, "/")
	  pRotes := strings.Split(path, "/")
      
	  for j, v := range partsRoute {
		if v != "" {
		  f := v[0:1]
		  l := v[len(v)-1:]
		  if f == "{" && l == "}" {
			mp[v[1:len(v)-1]] = pRotes[j]
			flag = true
		  } else if pRotes[j] != v {
  
			strs := strings.Split(v, "{")
			if len(strs) > 0 {
			  key := strs[1][:len(strs[1])-1]
			  mp[key] = pRotes[j][len(strs[0]):]
			  flag = true
			} else {
			  flag = false
			  break
			}
		  }
		  flag = true
		}
	  }
	  if flag {
		if hr, found := s.handlers[route]; found {
		  return mp, hr
		}
		break
	  }
	}
  
	return nil, nil
  
  }