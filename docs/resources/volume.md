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
  name = "data-01"
  size = 100 # GB. 늘리면 os-extend, 줄이면 오류

  # usage_plan_type = "hourly"  # "hourly" | "monthly" (기본값 monthly)
  # bootable        = true      # 부팅 가능 볼륨 여부, 기본값 false
}
```

## Schema

### Required

- `size` (Number) 볼륨 크기(GB). 값을 늘리면 온라인 확장(`os-extend`)을 시도하며, 줄이면 오류가 발생합니다.

### Optional

- `name` (String) 볼륨 이름.
- `description` (String) 볼륨 설명.
- `volume_type` (String) 볼륨 타입. 변경 시 **재생성**.
- `availability_zone` (String) 가용 영역. 변경 시 **재생성**.
- `snapshot_id` (String) 스냅샷으로부터 복원할 경우의 스냅샷 ID. 변경 시 **재생성**.
- `usage_plan_type` (String) 과금 단위. `hourly` 또는 `monthly`(kt cloud 기본값 `monthly`). 생성 요청에만 반영되며, kt cloud 조회 API 응답에 이 필드가 없어서 조회 시 값을 다시 가져오지는 않습니다. 변경 시 **재생성**.
- `bootable` (Boolean) 부팅 가능한 볼륨으로 생성할지 여부. 기본값 `false`. 조회 시 kt cloud 응답을 기준으로 갱신됩니다. 변경 시 **재생성**.

### Read-Only

- `id` (String) 볼륨 UUID.
- `status` (String) 볼륨 상태 (`available`, `in-use` 등).
- `created_at` (String) 생성 시각.

## Import

```shell
terraform import ktcloud_volume.data <VOLUME-UUID>
```
