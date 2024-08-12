package ws

import (
	"log"
)

type HandlerFunc func(*Context)

// 存储路径对应回调函数
type Servlet struct {
	wsMethods map[string]map[string]HandlerFunc
}

func NewServelet() *Servlet {
	return &Servlet{
		wsMethods: make(map[string]map[string]HandlerFunc),
	}
}

// 给对应路径注册业务函数
func (s *Servlet) use(method string, path string, f HandlerFunc) {
	log.Printf("registe websocekt callfunc [%s] [%s]\n", method, path)
	_, ok := s.wsMethods[method]
	if !ok {
		s.wsMethods[method] = make(map[string]HandlerFunc)
	}
	s.wsMethods[method][path] = f
}

// 获取对应路径注册的回调函数
func (s *Servlet) getFunc(method string, path string) (f HandlerFunc, ok bool) {
	v, ok := s.wsMethods[method]
	if !ok {
		log.Printf("this [%s] method does not exist\n", method)
		return nil, ok
	}
	f, ok = v[path]
	if !ok {
		log.Printf("this [%s] route does not exist\n", path)
	}
	return f, ok
}
