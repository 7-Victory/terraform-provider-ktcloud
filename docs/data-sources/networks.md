---
page_title: "ktcloud_networks Data Source - terraform-provider-ktcloud"
subcategory: ""
description: |-
  사용 가능한 네트워크(Tier) 목록을 조회합니다.
---

# ktcloud_networks (Data Source)

사용 가능한 네트워크(Tier) 목록을 조회합니다.

kt cloud "Tier 정보 조회" API(`GET /nsm/v1/network`)를 사용하며, 응답이 여러 페이지로
나뉘어 있어도 전부 모아서 반환합니다. 외부 연결(UNTRUST) 타입을 포함한 전체 결과를
가져오기 위해 내부적으로 `networkType=ALL`로 조회합니다.

~> **Note** 일부 항목(외부 업링크 등, `type`이 `"ENT"`인 경우)은 `id`(refId)가 아예 없어
빈 문자열로 나옵니다. **VM에 연결할 네트워크를 고를 땐 `id`가 비어있지 않은 항목만
사용하세요** — 비어있는 걸 `ktcloud_server.networks.uuid`에 넣으면 실패합니다.

## Example Usage

```terraform
data "ktcloud_networks" "all" {}

data "ktcloud_networks" "dmz" {
  name_contains = "DMZ" # 대소문자 무시 부분 일치. 생략하면 전체
}

resource "ktcloud_server" "web" {
  # ...
  networks {
    uuid = data.ktcloud_networks.dmz.networks[0].id # refId. network_id 아님!
  }
}
```

## Schema

### Optional

- `name_contains` (String) 이름에 이 문자열이 포함된 네트워크만 반환합니다 (대소문자 무시).

### Read-Only

- `networks` (Attributes List) 조회된 네트워크 목록. (see [below for nested schema](#nestedatt--networks))

<a id="nestedatt--networks"></a>
### Nested Schema for `networks`

Read-Only:

- `id` (String) **`ktcloud_server.networks.uuid`에 써야 하는 값(refId)입니다.**
- `network_id` (String) kt cloud 자체 네트워크 관리 ID. **`networks.uuid`에 이 값을 쓰면 `Network ... could not be found` 에러가 납니다** — 콘솔 등에서 대조용으로만 참고하세요.
- `name` (String) 네트워크(Tier) 이름.
- `cidr` (String) CIDR 대역.
- `gateway_ip` (String) 게이트웨이 IP.
- `type` (String) 네트워크 타입.
- `status` (String) 상태 (`ACTIVE` 등).
- `shared` (Boolean) 공유 네트워크 여부.
