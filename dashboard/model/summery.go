package model

import (
	"git.henghajiang.com/backend/api_gateway_v2/core/routing"
)

type SummeryModel struct {
	Table *routing.Table
}

type SummeryResp struct {
	Table       *routing.Table   `json:"table"`
	Route       *RouteInfo       `json:"route"`
	Service     *ServiceInfo     `json:"service"`
	Node        *NodeInfo        `json:"node"`
	HealthCheck *HealthCheckInfo `json:"health_check"`
}

func (m *SummeryModel) GetSummery() (*SummeryResp, error) {
	//var err error
	res := &SummeryResp{}

	res.Table = m.Table

	return res, nil
}
