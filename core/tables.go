package core

import "sync"

type RouteTableMap struct {
	sync.RWMutex
	internal map[string]*Router
}

type OnlineRouteTableMap struct {
	sync.RWMutex
	internal map[string]*Router
}

type ServerTableMap struct {
	sync.RWMutex
	internal map[string]*Server
}

func NewRouteTableMap() *RouteTableMap {
	return &RouteTableMap{
		internal: make(map[string]*Router),
	}
}

func (m *RouteTableMap) Load(key string) (value *Router, ok bool) {
	m.RLock()
	value, ok = m.internal[key]
	m.RUnlock()
	return value, ok
}

func (m *RouteTableMap) Delete(key string) {
	m.Lock()
	delete(m.internal, key)
	m.Unlock()
}

func (m *RouteTableMap) Store(key string, value *Router) {
	m.Lock()
	m.internal[key] = value
	m.Unlock()
}

func NewOnlineRouteTableMap() *OnlineRouteTableMap {
	return &OnlineRouteTableMap{
		internal: make(map[string]*Router),
	}
}

func (m *OnlineRouteTableMap) Load(key string) (value *Router, ok bool) {
	m.RLock()
	value, ok = m.internal[key]
	m.RUnlock()
	return value, ok
}

func (m *OnlineRouteTableMap) Delete(key string) {
	m.Lock()
	delete(m.internal, key)
	m.Unlock()
}

func (m *OnlineRouteTableMap) Store(key string, value *Router) {
	m.Lock()
	m.internal[key] = value
	m.Unlock()
}

func NewServerTableMap() *ServerTableMap {
	return &ServerTableMap{
		internal: make(map[string]*Server),
	}
}

func (m *ServerTableMap) Load(key string) (value *Server, ok bool) {
	m.RLock()
	value, ok = m.internal[key]
	m.RUnlock()
	return value, ok
}

func (m *ServerTableMap) Delete(key string) {
	m.Lock()
	delete(m.internal, key)
	m.Unlock()
}

func (m *ServerTableMap) Store(key string, value *Server) {
	m.Lock()
	m.internal[key] = value
	m.Unlock()
}
