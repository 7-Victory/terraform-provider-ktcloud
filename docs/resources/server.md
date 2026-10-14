---
page_title: "ktcloud_server Resource - terraform-provider-ktcloud"
subcategory: ""
description: |-
  kt cloud 가상 서버(VM)를 관리합니다.
---

# ktcloud_server (Resource)

kt cloud 가상 서버(VM)를 관리합니다.

## Example Usage

```terraform
resource "ktcloud_server" "web" {
  name              = "web-01"
  flavor_id         = data.ktcloud_flavors.small.flavors[0].id
  image_id          = data.ktcloud_images.rocky.images[0].id
  keypair_name      = ktcloud_keypair.demo.name
  availability_zone = "DX-M1"
  root_volume_size  = 50
  root_volume_type  = "SSD" # "SSD"→SSD-2000iops, "HDD"→HDD-2000iops. 생략 시 HDD-2000iops

  networks {
    uuid     = var.network_id # /nsm/v1/network 응답의 refId (networkId 아님)
    fixed_ip = "172.25.0.11"  # 생략 시 자동 할당
  }

  user_data = file("cloud-init.sh") # 평문 입력 → Provider가 base64 인코딩

  metadata = {
    managed_by = "terraform"
  }
}
```

## Schema

### Required

- `name` (String) VM 이름. 변경 시 재생성 없이 이름만 수정됩니다.
- `flavor_id` (String) VM 스펙 ID. `ktcloud_flavors` 데이터소스로 조회할 수 있습니다. 변경 시 재생성 없이 resize API로 처리됩니다 (kt cloud 쪽 사정으로 재부팅이 발생할 수 있습니다).
- `image_id` (String) OS 이미지 ID. `ktcloud_images` 데이터소스로 조회할 수 있습니다. 변경 시 **재생성**.
- `keypair_name` (String) SSH 키페어 이름. kt cloud API상 필수값입니다(Null 불가). 이미 계정에 있는 키 이름을 문자열로 그대로 써도 되고, `ktcloud_keypair` 리소스를 참조해도 됩니다. 변경 시 **재생성**.

### Optional

- `availability_zone` (String) 가용 영역. 생략하면 `"DX-M1"`(d1 zone 기준)으로 기본 설정됩니다. **d1이 아닌 zone(d2/gd1/gd4)을 쓰신다면 이 기본값이 틀릴 수 있으니 반드시 직접 지정하세요.** 변경 시 **재생성**.
- `user_data` (String) cloud-init 스크립트. 평문으로 입력하면 Provider가 base64로 인코딩해 전송합니다. 변경 시 **재생성**.
- `root_volume_size` (Number) 루트 볼륨 크기(GB). kt cloud API는 **50(Linux) 또는 100(Windows)만 허용**하며, 그 외 값은 plan 단계에서 에러로 막힙니다. 생략하면 `50`(Linux 이미지 기준)으로 기본 설정됩니다 — **Windows 이미지라면 반드시 100으로 직접 지정하세요.** 변경 시 **재생성**.
- `root_volume_type` (String) 루트 볼륨 타입. `root_volume_size`와 함께 사용합니다. `"SSD"`/`"HDD"` 축약형은 각각 `SSD-2000iops`/`HDD-2000iops`로 자동 변환되며, 그 외 값은 `GET /volume/{project_id}/types` 응답의 `name`을 그대로 씁니다. 생략하면 `HDD-2000iops`가 기본값입니다. kt cloud 공식 서버 생성 가이드엔 이 필드가 없지만, 실제로 지정한 타입대로 볼륨이 생성됨을 확인했습니다. 변경 시 **재생성**.
- `metadata` (Map of String) VM 메타데이터 key/value. 변경 시 **재생성**.
- `networks` (Block List) 연결할 네트워크(Tier) 목록. 순서가 NIC 순서를 결정합니다. 변경 시 **재생성**. (see [below for nested schema](#nestedblock--networks))

### Read-Only

- `id` (String) VM UUID.
- `status` (String) VM 상태 (`ACTIVE`, `SHUTOFF` 등).
- `private_ip` (String) 사설 IP.
- `public_ip` (String) 공인(floating) IP. 할당된 경우에만 값이 있습니다.
- `created_at` (String) 생성 시각.

<a id="nestedblock--networks"></a>
### Nested Schema for `networks`

Required:

- `uuid` (String) 네트워크(Tier) UUID. **`networkId`가 아니라 `refId` 값을 써야 합니다.** `networkId`를 넣으면 `Network ... could not be found` 400 에러가 납니다. [`ktcloud_networks`](../data-sources/networks.md) 데이터소스로 조회하면 `id` 필드가 바로 `refId`입니다.

Optional:

- `fixed_ip` (String) 지정할 사설 IP. 생략하면 자동 할당됩니다.

## Import

```shell
terraform import ktcloud_server.web <VM-UUID>
```

> import 직후에는 `networks` / `user_data`가 state에 없습니다. `terraform plan`이 재생성을 제안하면, HCL에 실제 값을 채워 넣거나 해당 블록을 `lifecycle { ignore_changes = [...] }`로 무시하세요.
