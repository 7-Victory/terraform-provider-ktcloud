---
page_title: "ktcloud_volume Resource - terraform-provider-ktcloud"
subcategory: ""
description: |-
  kt cloud 블록 스토리지 볼륨을 관리합니다.
---

# ktcloud_volume (Resource)

kt cloud 블록 스토리지 볼륨을 관리합니다.

## Example Usage

```terraform
resource "ktcloud_volume" "data" {
  name        = "data-01"
  size        = 100  # GB. 늘리면 os-extend, 줄이면 오류
  volume_type = "SSD" # "SSD"/"HDD" 는 size 에 맞는 IOPS 등급으로 자동 변환. 생략 시 "HDD"

  # usage_plan_type = "hourly" # "hourly" | "monthly"(기본값). 단기 테스트면 hourly로 비용 절감
  # bootable      = true     # 부팅 가능 볼륨 여부, 기본값 false
}
```

## Schema

### Required

- `size` (Number) 볼륨 크기(GB). 값을 늘리면 온라인 확장(`os-extend`)을 시도하며, 줄이면 오류가 발생합니다. kt cloud 공식 가이드엔 "300GB 초과 시 400/500/1000/2000만 가능"이라고 돼 있지만, 실제로는 `850`처럼 그 목록에 없는 값도 정상 생성됨을 확인했습니다(강제 아님).

### Optional

- `name` (String) 볼륨 이름.
- `description` (String) 볼륨 설명.
- `volume_type` (String) 볼륨 타입. `"SSD"`/`"HDD"` 축약형을 쓰면 `size`에 맞는 구체적인 IOPS 등급으로 자동 변환되며(아래 표), 그 외 값은 `GET /volume/{project_id}/types` 응답의 `name`을 그대로 씁니다. 생략하면 `"HDD"`가 기본값입니다. 변경 시 **재생성**.

  kt cloud가 볼륨 크기에 따라 IOPS 등급을 다르게 배정하기 때문에(공식 가이드
  `/home/cloud-user/terraform/volumeiops` 기준, 실 계정 100~700GB 전 구간 검증 완료),
  이 provider가 `size`를 보고 올바른 등급 문자열로 자동 치환합니다:

  | 크기 | SSD | HDD |
  |---|---|---|
  | ~100GB | `SSD-2000iops` | `HDD-2000iops` |
  | ~200GB | `SSD-4800iops` | `HDD-4800iops` |
  | ~300GB | `SSD-6400iops` | `HDD-5600iops` |
  | ~600GB | `SSD-6400iops`(400GB까지) / `SSD-8000iops`(500~600GB) | `HDD-6000iops`(400GB 이상 동일) |
  | 700GB~ | `SSD-9600iops` | `HDD-6000iops` |

  이 리소스는 `volume_type`을 조회로 재동기화하지 않으므로(`"SSD"`/`"HDD"` 축약형을 쓴
  config와 실제 kt cloud 값이 달라져서 매 plan마다 재생성을 시도하는 drift를 막기 위함),
  state에는 config에 적은 값이 그대로 남습니다 — 실제로 적용된 정확한 등급 문자열은
  콘솔이나 `GET /volume/{project_id}/volumes/{id}` 응답에서 확인하세요.
- `availability_zone` (String) 가용 영역. 변경 시 **재생성**.
- `snapshot_id` (String) 스냅샷으로부터 복원할 경우의 스냅샷 ID. 변경 시 **재생성**.
- `usage_plan_type` (String) 과금 단위. `hourly` 또는 `monthly`. 생략하면 `"monthly"`가 기본값입니다(kt cloud 자체 기본값과 동일). 단기 테스트용 볼륨이라 비용을 줄이고 싶다면 `"hourly"`를 직접 지정하세요. 생성 요청에만 반영되며, kt cloud 조회 API 응답에 이 필드가 없어서 조회 시 값을 다시 가져오지는 않습니다. 변경 시 **재생성**.
- `bootable` (Boolean) 부팅 가능한 볼륨으로 생성할지 여부. 기본값 `false`. 조회 시 kt cloud 응답을 기준으로 갱신됩니다. 변경 시 **재생성**.

### Read-Only

- `id` (String) 볼륨 UUID.
- `status` (String) 볼륨 상태 (`available`, `in-use` 등).
- `created_at` (String) 생성 시각.

## Import

```shell
terraform import ktcloud_volume.data <VOLUME-UUID>
```
