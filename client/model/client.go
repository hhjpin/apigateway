package model

import (
	"bytes"
	"fmt"
	"git.henghajiang.com/backend/api_gateway_v2/sdk/golang"
	"github.com/coreos/etcd/clientv3"
	"github.com/hhjpin/goutils/errors"
	"github.com/hhjpin/goutils/logger"
)

type ClientModel struct {
	Cl *clientv3.Client
}

type ClientRegisterReq struct {
	Service         string      `json:"service"`
	Port            int         `json:"port"`
	Host            string      `json:"host"`
	HealthCheckPath string      `json:"health_check_path"`
	Routes          []RouteItem `json:"routes"`
}

type RouteItem struct {
	Frontend string `json:"frontend"`
	Backend  string `json:"backend"`
	Method   string `json:"method"`
}

func (c *ClientModel) Register(r *ClientRegisterReq) error {
	hc := golang.NewHealthCheck(r.HealthCheckPath, 10, 5, 3, true)
	node := golang.NewNode(r.Host, r.Port, hc)
	svr := golang.NewService(r.Service, node)
	routes, err := c.newRoutes(r.Routes, svr)
	if err != nil {
		return err
	}
	gw := golang.NewApiGatewayRegistrant(c.Cl, node, svr, routes)
	if err := gw.Register(); err != nil {
		return err
	}
	return nil
}

func (c *ClientModel) newRoutes(routes []RouteItem, svr *golang.Service) ([]*golang.Router, error) {
	var rts []*golang.Router
	for _, rt := range routes {
		if rt.Method == "" || rt.Frontend == "" || rt.Backend == "" {
			err := errors.NewFormat(9, fmt.Sprintf("route is error: %#v", rt))
			logger.Error(err)
			return nil, err
		}
		name := bytes.Trim([]byte(rt.Frontend), "/")
		for i, b := range name {
			if b == '/' {
				name[i] = '+'
			}
		}
		nameStr := rt.Method + "@" + string(name)
		rts = append(rts, golang.NewRouter(nameStr, rt.Method, rt.Frontend, rt.Backend, svr))
	}
	return rts, nil
}
