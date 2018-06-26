package core

import "sync"

type RouteTableMap struct {
	sync.RWMutex
	internal map[FrontApiString]*Router
}

type OnlineRouteTableMap struct {
	sync.RWMutex
	internal map[FrontApiString]*Router
}

type ServerTableMap struct {
	sync.RWMutex
	internal map[ServerNameString]*Server
}

type ServiceTableMap struct {
	sync.RWMutex
	internal map[ServiceNameString]*Service
}

func NewRouteTableMap() *RouteTableMap {
	return &RouteTableMap{
		internal: make(map[FrontApiString]*Router),
	}
}

func (m *RouteTableMap) Load(key FrontApiString) (value *Router, ok bool) {
	m.RLock()
	value, ok = m.internal[key]
	m.RUnlock()
	return value, ok
}

func (m *RouteTableMap) Delete(key FrontApiString) {
	m.Lock()
	delete(m.internal, key)
	m.Unlock()
}

func (m *RouteTableMap) Store(key FrontApiString, value *Router) {
	m.Lock()
	m.internal[key] = value
	m.Unlock()
}

func NewOnlineRouteTableMap() *OnlineRouteTableMap {
	return &OnlineRouteTableMap{
		internal: make(map[FrontApiString]*Router),
	}
}

func (m *OnlineRouteTableMap) Load(key FrontApiString) (value *Router, ok bool) {
	m.RLock()
	value, ok = m.internal[key]
	m.RUnlock()
	return value, ok
}

func (m *OnlineRouteTableMap) Delete(key FrontApiString) {
	m.Lock()
	delete(m.internal, key)
	m.Unlock()
}

func (m *OnlineRouteTableMap) Store(key FrontApiString, value *Router) {
	m.Lock()
	m.internal[key] = value
	m.Unlock()
}

func NewServerTableMap() *ServerTableMap {
	return &ServerTableMap{
		internal: make(map[ServerNameString]*Server),
	}
}

func (m *ServerTableMap) Load(key ServerNameString) (value *Server, ok bool) {
	m.RLock()
	value, ok = m.internal[key]
	m.RUnlock()
	return value, ok
}

func (m *ServerTableMap) Delete(key ServerNameString) {
	m.Lock()
	delete(m.internal, key)
	m.Unlock()
}

func (m *ServerTableMap) Store(key ServerNameString, value *Server) {
	m.Lock()
	m.internal[key] = value
	m.Unlock()
}

func (m *ServerTableMap) Range(f func(key ServerNameString, value *Server)) {
	m.RLock()
	for k, v := range m.internal {
		f(k, v)
	}
	m.RUnlock()
}

func (m *ServerTableMap) Len() (length int) {
	m.RLock()
	length = len(m.internal)
	m.RUnlock()
	return length
}

func NewServiceTableMap() *ServiceTableMap {
	return &ServiceTableMap{
		internal: make(map[ServiceNameString]*Service),
	}
}

func (m *ServiceTableMap) Load(key ServiceNameString) (value *Service, ok bool) {
	m.RLock()
	value, ok = m.internal[key]
	m.RUnlock()
	return value, ok
}

func (m *ServiceTableMap) Delete(key ServiceNameString) {
	m.Lock()
	delete(m.internal, key)
	m.Unlock()
}

func (m *ServiceTableMap) Store(key ServiceNameString, value *Service) {
	m.Lock()
	m.internal[key] = value
	m.Unlock()
}

