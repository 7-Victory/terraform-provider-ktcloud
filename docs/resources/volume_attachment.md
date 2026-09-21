---
page_title: "ktcloud_volume_attachment Resource - terraform-provider-ktcloud"
subcategory: ""
description: |-
  볼륨을 VM에 연결합니다.
---

# ktcloud_volume_attachment (Resource)

볼륨을 VM에 연결합니다.

## Example Usage

```terraform
resource "ktcloud_volume_attachment" "data" {
  server_id = ktcloud_server.web.id
  volume_id = ktcloud_volume.data.id
  # device  = "/dev/vdb"   # 생략 시 자동 할당
}
```

## Schema

### Required

- `server_id` (String) VM UUID. 변경 시 **재생성**.
- `volume_id` (String) 볼륨 UUID. 변경 시 **재생성**.

### Optional

- `device` (String) 게스트 OS 상의 디바이스 경로 (예: `/dev/vdb`). 생략하면 자동 할당됩니다. 변경 시 **재생성**.

### Read-Only

- `id` (String) `<server_id>/<volume_id>` 형식의 복합 ID.

## Import

```shell
terraform import ktcloud_volume_attachment.data <SERVER-UUID>/<VOLUME-UUID>
```
