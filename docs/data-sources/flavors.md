---
page_title: "ktcloud_flavors Data Source - terraform-provider-ktcloud"
subcategory: ""
description: |-
  사용 가능한 VM 스펙(flavor) 목록을 조회합니다.
---

# ktcloud_flavors (Data Source)

사용 가능한 VM 스펙(flavor) 목록을 조회합니다.

## Example Usage

```terraform
data "ktcloud_flavors" "small" {
  name_contains = "2x4" # 예: 2vCore 4GB. 대소문자 무시 부분 일치. 생략하면 전체
}

output "available_flavors" {
  value = data.ktcloud_flavors.small.flavors
}
```

`name_contains`는 정확히 일치가 아니라 **부분 문자열 포함** 필터입니다. 정확히 이름이 같은 것 하나만 골라야 한다면:

```terraform
locals {
  exact = [for f in data.ktcloud_flavors.small.flavors : f if f.name == "2x4"][0]
}
```

## Schema

### Optional

- `name_contains` (String) 이름에 이 문자열이 포함된 스펙만 반환합니다 (대소문자 무시).

### Read-Only

- `flavors` (Attributes List) 조회된 스펙 목록. (see [below for nested schema](#nestedatt--flavors))

<a id="nestedatt--flavors"></a>
### Nested Schema for `flavors`

Read-Only:

- `id` (String) 스펙 ID.
- `name` (String) 스펙 이름.
- `vcpus` (Number) vCPU 수.
- `ram` (Number) 메모리(MB).
- `disk` (Number) 디스크(GB).
