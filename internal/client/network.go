package client

import (
	"context"
	"fmt"
	"net/http"
)

// Network 는 kt cloud 네트워크(Tier) 정보입니다 ("Tier 정보 조회" API 응답 기준).
// 응답에 더 많은 필드가 있지만 쓸모 있는 것만 담습니다. 외부 업링크 등 일부 항목은
// RefID/CIDR/GatewayIP/Status 가 비어 있을 수 있습니다 (VM 연결용으로 쓸 수 없음).
type Network struct {
	// RefID 는 ktcloud_server.networks.uuid 에 써야 하는 값입니다.
	// NetworkID(아래)와는 다른 값이니 혼동하지 마세요.
	RefID       string `json:"refId"`
	NetworkID   string `json:"networkId"` // kt cloud 자체 관리용 ID. networks.uuid 에 쓰면 404 남
	NetworkName string `json:"networkName"`
	RefName     string `json:"refName"`
	CIDR        string `json:"cidr"`
	GatewayIP   string `json:"gatewayIp"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Shared      bool   `json:"shared"`
}

type networksPagination struct {
	Size   int `json:"size"`
	Total  int `json:"total"`
	Offset int `json:"offset"`
}

type networksResponse struct {
	Pagination networksPagination `json:"pagination"`
	Data       []Network          `json:"data"`
}

// ListNetworks 는 계정의 네트워크(Tier) 목록을 조회합니다.
//
// 공식 API 가이드(/home/cloud-user/terraform/tier) 기준:
//   - 페이지네이션은 "page"(1부터 시작) / "size"(기본 20) 쿼리 파라미터로 제어합니다
//     (응답에 담긴 "offset" 은 현재 페이지 시작 위치를 알려주는 정보용 필드일 뿐,
//     요청 파라미터가 아닙니다).
//   - networkType 기본값이 "trust" 라, 지정하지 않으면 UNTRUST(외부 연결) Tier 가
//     조용히 빠질 수 있습니다. 항상 "ALL" 을 명시해서 전체를 가져옵니다.
func (c *Client) ListNetworks(ctx context.Context) ([]Network, error) {
	var all []Network

	for page := 1; page <= 25; page++ {
		path := fmt.Sprintf("%s?networkType=ALL&page=%d&size=20", PathNetworks, page)

		var out networksResponse
		if err := c.Do(ctx, http.MethodGet, ServiceNetwork, path, nil, &out); err != nil {
			return nil, err
		}
		if len(out.Data) == 0 {
			break
		}

		// page 파라미터가 무시돼 같은 첫 항목이 반복되면 중단합니다 (안전장치).
		if len(all) > 0 && out.Data[0].RefID == all[0].RefID {
			break
		}

		all = append(all, out.Data...)

		if out.Pagination.Total <= 0 || len(all) >= out.Pagination.Total || len(out.Data) < out.Pagination.Size {
			break
		}
	}

	return all, nil
}
