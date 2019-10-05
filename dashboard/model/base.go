package model

import "git.henghajiang.com/backend/golang_utils/log"

var logger = log.Logger

type Status int

const (
	Offline Status = iota
	Online
	BreakDown
)

type Node struct {
	Id     string   `json:"id"`
	Name   string   `json:"name"`
	Host   string   `json:"host"`
	Port   int      `json:"port"`
	Status Status   `json:"status"`
	Attr   []string `json:"attr"`
}

type Service struct {
	Name string   `json:"name"`
	Node []*Node  `json:"node"` //todo
	Attr []string `json:"attr"`
}

type Route struct {
	Id       string   `json:"id"`
	Name     string   `json:"name"`
	Status   Status   `json:"status"`
	Frontend string   `json:"frontend"`
	Backend  string   `json:"backend"`
	Service  Service  `json:"service"`
	Attr     []string `json:"attr"`
}

type HealthCheck struct {
	Id        string   `json:"id"`
	Path      string   `json:"path"`
	Timeout   int      `json:"timeout"`
	Interval  int      `json:"interval"`
	Retry     bool     `json:"retry"`
	RetryTime int      `json:"retry_time"`
	Attr      []string `json:"attr"`
}

type RouteInfo struct {
	Total int      `json:"total"`
	List  []*Route `json:"list"`
}

type ServiceInfo struct {
	Total int        `json:"total"`
	List  []*Service `json:"list"`
}

type NodeInfo struct {
	Total int     `json:"total"`
	List  []*Node `json:"list"`
}

type HealthCheckInfo struct {
	Total int            `json:"total"`
	List  []*HealthCheck `json:"list"`
}
