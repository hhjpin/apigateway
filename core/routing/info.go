package routing

type RouteInfo struct {
	Name        string            `json:"name"`
	Status      Status            `json:"status"`
	FrontendApi string            `json:"frontend_api"`
	BackendApi  string            `json:"backend_api"`
	Service     ServiceNameString `json:"service"`
}

type ServiceInfo struct {
	Name             string               `json:"name"`
	Status           Status               `json:"status"`
	Endpoint         []EndpointNameString `json:"endpoint"`
	AcceptHttpMethod []string             `json:"accept_http_method"`
}

type HealthCheckInfo struct {
	Id        string `json:"id"`
	Path      string `json:"path"`
	Timeout   uint8  `json:"timeout"`
	Interval  uint8  `json:"interval"`
	Retry     bool   `json:"retry"`
	RetryTime uint8  `json:"retry_time"`
}

type EndpointInfo struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Host        string           `json:"host"`
	Port        int              `json:"port"`
	Status      Status           `json:"status"`
	HealthCheck *HealthCheckInfo `json:"health_check"`
}

type TableInfo struct {
	Version       string                               `json:"version"`
	ServiceTable  map[ServiceNameString]*ServiceInfo   `json:"service_table"`
	EndpointTable map[EndpointNameString]*EndpointInfo `json:"endpoint_table"`
	RouterTable   map[RouterNameString]*RouteInfo      `json:"router_table"`
}

func (r *Table) GetTableInfo() *TableInfo {
	t := &TableInfo{
		Version:       r.Version,
		ServiceTable:  map[ServiceNameString]*ServiceInfo{},
		EndpointTable: map[EndpointNameString]*EndpointInfo{},
		RouterTable:   map[RouterNameString]*RouteInfo{},
	}

	r.endpointTable.Range(func(k EndpointNameString, v *Endpoint) bool {
		t.EndpointTable[k] = &EndpointInfo{
			ID:     v.id,
			Name:   string(v.name),
			Host:   string(v.host),
			Port:   v.port,
			Status: v.status,
		}
		if v.healthCheck != nil {
			t.EndpointTable[k].HealthCheck = &HealthCheckInfo{
				Id:        v.healthCheck.id,
				Path:      string(v.healthCheck.path),
				Timeout:   v.healthCheck.timeout,
				Interval:  v.healthCheck.interval,
				Retry:     v.healthCheck.retry,
				RetryTime: v.healthCheck.retryTime,
			}
		}
		return false
	})

	r.serviceTable.Range(func(k ServiceNameString, v *Service) bool {
		t.ServiceTable[k] = &ServiceInfo{
			Name:             string(v.name),
			Status:           Offline,
			Endpoint:         []EndpointNameString{},
			AcceptHttpMethod: []string{},
		}
		for _, method := range v.acceptHttpMethod {
			t.ServiceTable[k].AcceptHttpMethod = append(t.ServiceTable[k].AcceptHttpMethod, string(method))
		}
		v.ep.Range(func(k1 EndpointNameString, k2 *Endpoint) bool {
			t.ServiceTable[k].Endpoint = append(t.ServiceTable[k].Endpoint, k1)
			if k2.status == Online {
				t.ServiceTable[k].Status = Online
			}
			return false
		})
		return false
	})

	r.routerTable.Range(func(k RouterNameString, v *Router) {
		t.RouterTable[k] = &RouteInfo{
			Name:        string(v.name),
			Status:      v.status,
			FrontendApi: string(v.frontendApi.path),
			BackendApi:  string(v.backendApi.path),
			Service:     v.service.nameString,
		}
	})

	return t
}
