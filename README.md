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
| Data Source | `ktcloud_networks` | 네트워크(Tier) 목록 |

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
| `flavor_id` | string, 필수 | 변경 시 재생성 없이 **resize API**로 처리 (resize → confirmResize) |
| `image_id` | string, 필수 | 변경 시 **재생성** |
| `keypair_name` | string, 필수 | kt cloud API 상 Null 불가. 변경 시 **재생성** |
| `availability_zone` | string | 생략 시 기본값 `"DX-M1"`(d1 zone 기준). 변경 시 **재생성** |
| `user_data` | string | 평문 입력. 변경 시 **재생성** |
| `root_volume_size` | number | GB. kt cloud API 상 **50(Linux) 또는 100(Windows)만 허용**. 생략 시 기본값 `50` |
| `root_volume_type` | string | `root_volume_size`와 함께 사용. `"SSD"`/`"HDD"` 축약형은 `root_volume_size`에 맞는 IOPS 등급으로 자동 변환되고(`ktcloud_volume.volume_type` 참고), 그 외엔 `GET /volume/{project_id}/types` 응답의 `name` 값을 그대로 씀. 생략 시 기본값 `"HDD"`. kt cloud 공식 서버 생성 가이드엔 없는 필드지만 실제로 반영됨을 확인(2026-10-14) |
| `metadata` | map(string) | 변경 시 **재생성** |
| `networks` | block × N | `uuid`(필수), `fixed_ip`(선택). 변경 시 **재생성** |

> ⚠️ **`networks.uuid` 는 kt cloud 자체 네트워크 관리 ID(`networkId`)가 아니라
> `refId` 값을 써야 합니다.** `networkId` 를 넣으면 `Network ... could not be found`
> 400 에러가 납니다. 아래 `ktcloud_networks` 데이터소스로 조회하면 `id` 필드가 바로
> `refId` 입니다.
>
> `availability_zone` 을 생략하면 provider 가 `"DX-M1"`을 기본값으로 채웁니다. **이건 d1 zone
> 전용 기본값**이라, `zone`을 `d2`/`gd1`/`gd4`로 쓰신다면 반드시 `availability_zone`을
> 직접 지정하세요. (예전엔 생략 시 `500 Internal server error`가 났었는데, 이제 기본값이
> 채워져서 d1 에서는 생략해도 됩니다.)

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
  name        = "data-01"
  size        = 100          # GB. 늘리면 os-extend, 줄이면 오류
  volume_type = "SSD"         # "SSD"/"HDD" 축약형은 size 에 맞는 IOPS 등급으로 자동 변환. 생략 시 HDD

  # usage_plan_type = "hourly"  # "hourly" | "monthly"(기본값). 단기 테스트면 hourly로 비용 절감
  # bootable      = true      # 부팅 가능 볼륨 여부, 기본값 false. 미검증 옵션
}

resource "ktcloud_volume_attachment" "data" {
  server_id = ktcloud_server.web.id
  volume_id = ktcloud_volume.data.id
  # device  = "/dev/vdb"   # 생략 시 자동 할당
}
```

> ⚠️ `size`는 가이드상 "300GB 초과 시 400/500/1000/2000만 가능"이라고 돼 있지만 실제로는
> `850` 같은 값도 생성됩니다(강제 아님).
>
> **`volume_type = "SSD"`/`"HDD"`는 `size`에 맞는 구체적인 IOPS 등급으로 자동 변환됩니다**
> (공식 가이드 `/home/cloud-user/terraform/volumeiops` 기준, 실 계정 100~700GB 전 구간 검증 완료):
>
> | 크기 | SSD | HDD |
> |---|---|---|
> | ~100GB | 2,000 IOPS | 2,000 IOPS |
> | ~200GB | 4,800 IOPS | 4,800 IOPS |
> | ~300GB | 6,400 IOPS | 5,600 IOPS |
> | ~400GB | 6,400 IOPS | 6,000 IOPS (400GB 이상 동일) |
> | ~600GB | 8,000 IOPS | 6,000 IOPS |
> | 700GB~ | 9,600 IOPS | 6,000 IOPS |
>
> `volume_type`을 생략하면 기본값은 `"HDD"`(size에 맞는 등급으로 자동 변환)입니다.
> 이 리소스는 `volume_type`을 조회로 재동기화하지 않아 state에는 config에 적은 값
> (`"SSD"`/`"HDD"` 등) 그대로 남고, 실제 적용된 정확한 등급 문자열은 콘솔에서 확인하세요.

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
# (내부적으로 limit=1000 고정 조회 — 안 그러면 kt cloud API가 일부만 돌려줌)

data "ktcloud_networks" "dmz" {
  name_contains = "DMZ"     # 대소문자 무시 부분 일치. 생략하면 전체
}
# → networks[*].{ id(=refId, networks.uuid 에 쓸 값), network_id, name, cidr, gateway_ip, type, status, shared }
```

> ⚠️ 일부 항목(외부 업링크 등, `type = "ENT"`)은 `id`(refId)가 비어 있습니다. VM에
> 연결할 땐 `id`가 비어있지 않은 항목만 쓰세요.

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
