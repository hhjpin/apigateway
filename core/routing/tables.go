package routing

import (
	"sync"
	"sync/atomic"
)

type ApiRouterTableMap struct {
	sync.RWMutex
	internal map[FrontendApiString]*Router
}

type OnlineApiRouterTableMap struct {
	sync.RWMutex
	internal map[*FrontendApi]*Router
}

type EndpointTableMap struct {
	sync.RWMutex
	readNum  int32
	internal map[EndpointNameString]*Endpoint
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

func (m *ApiRouterTableMap) Range(f func(key FrontendApiString, value *Router)) {
	m.RLock()
	for k, v := range m.internal {
		f(k, v)
	}
	m.RUnlock()
}

func (m *ApiRouterTableMap) unsafeRange(f func(key FrontendApiString, value *Router)) {
	for k, v := range m.internal {
		f(k, v)
	}
}

func NewOnlineRouteTableMap() *OnlineApiRouterTableMap {
	return &OnlineApiRouterTableMap{
		internal: make(map[*FrontendApi]*Router),
	}
}

func (m *OnlineApiRouterTableMap) Load(key *FrontendApi) (value *Router, ok bool) {
	m.RLock()
	value, ok = m.internal[key]
	m.RUnlock()
	return value, ok
}

func (m *OnlineApiRouterTableMap) Delete(key *FrontendApi) {
	m.Lock()
	delete(m.internal, key)
	m.Unlock()
}

func (m *OnlineApiRouterTableMap) Store(key *FrontendApi, value *Router) {
	m.Lock()
	m.internal[key] = value
	m.Unlock()
}

func (m *OnlineApiRouterTableMap) Range(f func(key *FrontendApi, value *Router) bool) {
	m.RLock()
	for k, v := range m.internal {
		ret := f(k, v)
		if ret {
			break
		}
	}
	m.RUnlock()
}

func NewEndpointTableMap() *EndpointTableMap {
	return &EndpointTableMap{
		internal: make(map[EndpointNameString]*Endpoint),
	}
}

func (m *EndpointTableMap) readLock() {
	if atomic.AddInt32(&m.readNum, 1) == 1 {
		m.RLock()
	}
}

func (m *EndpointTableMap) readUnlock() {
	if atomic.AddInt32(&m.readNum, -1) == 0 {
		m.RUnlock()
	}
}

func (m *EndpointTableMap) Load(key EndpointNameString) (value *Endpoint, ok bool) {
	m.readLock()
	value, ok = m.internal[key]
	m.readUnlock()
	return value, ok
}

func (m *EndpointTableMap) Delete(key EndpointNameString) {
	m.Lock()
	delete(m.internal, key)
	m.Unlock()
}

func (m *EndpointTableMap) Store(key EndpointNameString, value *Endpoint) {
	m.Lock()
	m.internal[key] = value
	m.Unlock()
}

func (m *EndpointTableMap) Range(f func(key EndpointNameString, value *Endpoint) bool) {
	m.readLock()
	for k, v := range m.internal {
		ok := f(k, v)
		if ok {
			break
		}
	}
	m.readUnlock()
}

func (m *EndpointTableMap) unsafeRange(f func(key EndpointNameString, value *Endpoint)) {
	for k, v := range m.internal {
		f(k, v)
	}
}

func (m *EndpointTableMap) Len() (length int) {
	m.RLock()
	length = len(m.internal)
	m.RUnlock()
	return length
}

func (m *EndpointTableMap) equal(another *EndpointTableMap) (result bool) {
	m.readLock()
	another.readLock()
	result = cmpMap(m.internal, another.internal)
	another.readUnlock()
	m.readUnlock()
	return result
}

func cmpMap(m1, m2 map[EndpointNameString]*Endpoint) bool {
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

func (m *ServiceTableMap) Range(f func(key ServiceNameString, value *Service) bool) {
	m.RLock()
	for k, v := range m.internal {
		ret := f(k, v)
		if ret {
			break
		}
	}
	m.RUnlock()
}

func (m *ServiceTableMap) unsafeRange(f func(key ServiceNameString, value *Service)) {
	for k, v := range m.internal {
		f(k, v)
	}
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

func (m *RouterTableMap) Range(f func(key RouterNameString, value *Router)) {
	m.RLock()
	for k, v := range m.internal {
		f(k, v)
	}
	m.RUnlock()
}
