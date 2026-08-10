# terraform-provider-ktcloud

kt cloud **@D Platform** Open API 용 Terraform Provider 입니다.
[terraform-plugin-framework](https://github.com/hashicorp/terraform-plugin-framework) 기반, Protocol v6.

kt cloud @D Platform 의 Open API 는 RESTful 방식이며 OpenStack API 와 호환됩니다.
인증은 `POST /{zone}/identity/auth/tokens` 로 토큰을 발급받고, 이후 모든 요청의
`X-Auth-Token` 헤더에 그 토큰을 실어 보내는 구조입니다. 토큰 유효시간은 60분이며,
이 Provider 가 만료 5분 전과 `401` 응답 시 자동으로 재발급합니다.

> 참고 문서: <https://cloud.kt.com/docs/open-api-guide/d/guide/how-to-use>

---

## 지원 범위

| 종류 | 이름 | 설명 |
|---|---|---|
| Resource | `ktcloud_server` | 가상 서버(VM) |
| Resource | `ktcloud_keypair` | SSH 키페어 |
| Resource | `ktcloud_volume` | 블록 스토리지 볼륨 |
| Resource | `ktcloud_volume_attachment` | 볼륨 ↔ VM 연결 |
| Data Source | `ktcloud_flavors` | VM 스펙 목록 |
| Data Source | `ktcloud_images` | OS 이미지 목록 |

---

## Provider 설정

```hcl
provider "ktcloud" {
  zone     = "d1"       # 또는 KTCLOUD_ZONE
  username = "포털ID"    # 또는 KTCLOUD_USERNAME
  password = "비밀번호"   # 또는 KTCLOUD_PASSWORD
}
```

| 인자 | 필수 | 환경변수 | 기본값 | 설명 |
|---|---|---|---|---|
| `zone` | ✅ | `KTCLOUD_ZONE` | – | `d1`(DX-M1), `d2`(DX-Central), `gd1`(DX-G), `gd4`(DX-G-YS) |
| `username` | ✅ | `KTCLOUD_USERNAME` | – | 포털 사용자 ID |
| `password` | ✅ | `KTCLOUD_PASSWORD` | – | 포털 비밀번호 (sensitive) |
| `api_base` | | `KTCLOUD_API_BASE` | `https://api.ucloudbiz.olleh.com` | API 도메인 |
| `domain_id` | | `KTCLOUD_DOMAIN_ID` | `default` | 인증 domain id |
| `project_name` | | `KTCLOUD_PROJECT_NAME` | `username` 과 동일 | 인증 scope project name |
| `insecure` | | `KTCLOUD_INSECURE` | `false` | TLS 검증 생략 (테스트용) |
| `use_project_id_path` | | `KTCLOUD_USE_PROJECT_ID_PATH` | `false` | 요청 경로에 `project_id` 삽입 |
| `request_timeout` | | – | `60` | 개별 HTTP 요청 타임아웃(초) |

**자격증명은 HCL 에 하드코딩하지 말고 환경변수를 쓰세요.**

```bash
export KTCLOUD_USERNAME='your-id'
export KTCLOUD_PASSWORD='your-password'
export KTCLOUD_ZONE='d1'
```

---

## `ktcloud_server`

```hcl
resource "ktcloud_server" "web" {
  name             = "web-01"
  flavor_id        = data.ktcloud_flavors.small.flavors[0].id
  image_id         = data.ktcloud_images.rocky.images[0].id
  keypair_name     = ktcloud_keypair.demo.name
  root_volume_size = 50

  networks {
    uuid     = var.network_id
    fixed_ip = "172.25.0.11"   # 생략 시 자동 할당
  }

  user_data = file("cloud-init.sh")   # 평문 입력 → Provider 가 base64 인코딩
}
```

| 인자 | 타입 | 비고 |
|---|---|---|
| `name` | string, 필수 | 변경 시 재생성 없이 이름만 수정 |
| `flavor_id` | string, 필수 | 변경 시 **재생성** |
| `image_id` | string, 필수 | 변경 시 **재생성** |
| `keypair_name` | string | 변경 시 **재생성** |
| `availability_zone` | string | 변경 시 **재생성** |
| `user_data` | string | 평문 입력. 변경 시 **재생성** |
| `root_volume_size` | number | GB. 지정 시 `block_device_mapping_v2` 로 부팅 |
| `root_volume_type` | string | 위와 함께 사용 |
| `metadata` | map(string) | 변경 시 **재생성** |
| `networks` | block × N | `uuid`(필수), `fixed_ip`(선택). 변경 시 **재생성** |

> ⚠️ **`networks.uuid` 는 kt cloud 자체 네트워크 관리 API(`/nsm/v1/network`) 응답의
> `networkId` 가 아니라 `refId` 값을 써야 합니다.** `networkId` 를 넣으면
> `Network ... could not be found` 400 에러가 납니다. 콘솔에서 네트워크(Tier) 정보를
> 확인할 때 "네트워크 ID"로 표시되는 값과 실제 Nova 가 요구하는 UUID 가 다르니
> `/nsm/v1/network` 응답의 `refId` 필드를 사용하세요.
>
> `availability_zone` 도 실제로는 지정하는 걸 권장합니다 (예: d1 zone 이면
> `DX-M1`). 비워두면 API 게이트웨이가 `500 Internal server error` 를 낼 수 있습니다.

Computed: `id`, `status`, `private_ip`, `public_ip`, `created_at`

```bash
terraform import ktcloud_server.web <VM-UUID>
```

> import 직후에는 `networks` / `user_data` 가 state 에 없습니다. `terraform plan` 이
> 재생성을 제안하면, HCL 에 실제 값을 채워 넣거나 해당 블록을
> `lifecycle { ignore_changes = [...] }` 로 무시하세요.

---

## `ktcloud_keypair`

```hcl
resource "ktcloud_keypair" "demo" {
  name       = "tf-demo-key"
  public_key = file("~/.ssh/id_rsa.pub")   # 생략하면 kt cloud 가 새 키 생성
}
```

`public_key` 를 생략하면 `private_key` (sensitive) 에 개인키가 담깁니다.
**개인키가 tfstate 에 평문 저장되므로** state 를 암호화된 원격 백엔드에 두거나,
운영 환경에서는 로컬에서 만든 공개키를 등록하는 방식을 권장합니다.

```bash
terraform import ktcloud_keypair.demo <키페어이름>
```

---

## `ktcloud_volume` / `ktcloud_volume_attachment`

```hcl
resource "ktcloud_volume" "data" {
  name = "data-01"
  size = 100          # GB. 늘리면 os-extend, 줄이면 오류
}

resource "ktcloud_volume_attachment" "data" {
  server_id = ktcloud_server.web.id
  volume_id = ktcloud_volume.data.id
  # device  = "/dev/vdb"   # 생략 시 자동 할당
}
```

```bash
terraform import ktcloud_volume.data            <VOLUME-UUID>
terraform import ktcloud_volume_attachment.data <SERVER-UUID>/<VOLUME-UUID>
```

---

## Data Sources

```hcl
data "ktcloud_flavors" "small" {
  name_contains = "2x4"     # 대소문자 무시 부분 일치. 생략하면 전체
}
# → flavors[*].{ id, name, vcpus, ram, disk }

data "ktcloud_images" "rocky" {
  name_contains = "rocky"
}
# → images[*].{ id, name, status, min_disk, min_ram }
```

---

## 디버깅

```bash
TF_LOG=DEBUG terraform apply 2>&1 | tee tf.log
```

API 요청/응답 원문이 필요하면 `internal/client/client.go` 의 `roundTrip` 에
`tflog.Debug` 를 추가하세요. 오류 메시지에는 이미 요청 method/URL/HTTP status/응답 본문이
포함됩니다.

---

## 개발 · 기여

`GUIDE.md` 에 Rocky Linux 기준 전체 작업 절차(빌드 → 로컬 테스트 → git push → 배포)와
직접 수정해야 하는 파일 목록이 정리되어 있습니다.

## 라이선스

MPL-2.0 (HashiCorp Provider 관례). 필요에 따라 변경하세요.
