package model

import (
	"git.henghajiang.com/backend/api_gateway_v2/core/routing"
)

type SummeryModel struct {
	Table *routing.Table
}

type SummeryResp struct {
	Table *routing.TableInfo `json:"table"`
}

func (m *SummeryModel) GetSummery() (*SummeryResp, error) {
	res := &SummeryResp{
		Table: m.Table.GetTableInfo(),
	}
	return res, nil
}
