package core

import "sync"

type ApiRouterTableMap struct {
	sync.RWMutex
	internal map[FrontendApiString]*Router
}

type OnlineApiRouterTableMap struct {
	sync.RWMutex
	internal map[FrontendApiString]*Router
}

type ServerTableMap struct {
	sync.RWMutex
	internal map[ServerNameString]*Server
}

type ServiceTableMap struct {
	sync.RWMutex
	internal map[ServiceNameString]*Service
}

type RouterTableMap struct {
	sync.RWMutex
	internal map[RouterNameString]*Router
}

func NewApiRouterTableMap() *ApiRouterTableMap {
	return &ApiRouterTableMap{
		internal: make(map[FrontendApiString]*Router),
	}
}

func (m *ApiRouterTableMap) Load(key FrontendApiString) (value *Router, ok bool) {
	m.RLock()
	value, ok = m.internal[key]
	m.RUnlock()
	return value, ok
}

func (m *ApiRouterTableMap) Delete(key FrontendApiString) {
	m.Lock()
	delete(m.internal, key)
	m.Unlock()
}

func (m *ApiRouterTableMap) Store(key FrontendApiString, value *Router) {
	m.Lock()
	m.internal[key] = value
	m.Unlock()
}

func NewOnlineRouteTableMap() *OnlineApiRouterTableMap {
	return &OnlineApiRouterTableMap{
		internal: make(map[FrontendApiString]*Router),
	}
}

func (m *OnlineApiRouterTableMap) Load(key FrontendApiString) (value *Router, ok bool) {
	m.RLock()
	value, ok = m.internal[key]
	m.RUnlock()
	return value, ok
}

func (m *OnlineApiRouterTableMap) Delete(key FrontendApiString) {
	m.Lock()
	delete(m.internal, key)
	m.Unlock()
}

func (m *OnlineApiRouterTableMap) Store(key FrontendApiString, value *Router) {
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

func (m *ServerTableMap) equal(another *ServerTableMap) (result bool) {
	m.RLock()
	another.RLock()
	result = cmpMap(m.internal, another.internal)
	another.RUnlock()
	m.RUnlock()
	return result
}

func cmpMap(m1, m2 map[ServerNameString]*Server) bool {
	for k1, v1 := range m1 {
		if v2, has := m2[k1]; has {
			if v1.equal(v2) {
				return false
			}
		} else {
			return false
		}
	}
	for k2, v2 := range m2 {
		if v1, has := m1[k2]; has {
			if v1.equal(v2) {
				return false
			}
		} else {
			return false
		}
	}
	return true
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

func NewRouteTableMap() *RouterTableMap {
	return &RouterTableMap{
		internal: make(map[RouterNameString]*Router),
	}
}

func (m *RouterTableMap) Load(key RouterNameString) (value *Router, ok bool) {
	m.RLock()
	value, ok = m.internal[key]
	m.RUnlock()
	return value, ok
}

func (m *RouterTableMap) Delete(key RouterNameString) {
	m.Lock()
	delete(m.internal, key)
	m.Unlock()
}

func (m *RouterTableMap) Store(key RouterNameString, value *Router) {
	m.Lock()
	m.internal[key] = value
	m.Unlock()
}
