---
page_title: "ktcloud_images Data Source - terraform-provider-ktcloud"
subcategory: ""
description: |-
  사용 가능한 OS 이미지 목록을 조회합니다.
---

# ktcloud_images (Data Source)

사용 가능한 OS 이미지 목록을 조회합니다.

## Example Usage

```terraform
data "ktcloud_images" "rocky" {
  name_contains = "rocky" # 대소문자 무시 부분 일치. 생략하면 전체
}

output "available_images" {
  value = data.ktcloud_images.rocky.images
}
```

## Schema

### Optional

- `name_contains` (String) 이름에 이 문자열이 포함된 이미지만 반환합니다 (대소문자 무시).

### Read-Only

- `images` (Attributes List) 조회된 이미지 목록. (see [below for nested schema](#nestedatt--images))

<a id="nestedatt--images"></a>
### Nested Schema for `images`

Read-Only:

- `id` (String) 이미지 ID.
- `name` (String) 이미지 이름.
- `status` (String) 이미지 상태.
- `min_disk` (Number) 최소 디스크(GB).
- `min_ram` (Number) 최소 메모리(MB).
