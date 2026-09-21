---
page_title: "ktcloud Provider"
description: |-
  kt cloud @D Platform Open API 용 Terraform Provider.
---

# ktcloud Provider

kt cloud **@D Platform** Open API 용 Terraform Provider입니다. [terraform-plugin-framework](https://github.com/hashicorp/terraform-plugin-framework) 기반, Protocol v6로 동작합니다.

kt cloud @D Platform의 Open API는 RESTful 방식이며 OpenStack API와 호환됩니다. 인증은 `POST /{zone}/identity/auth/tokens`로 토큰을 발급받고, 이후 모든 요청의 `X-Auth-Token` 헤더에 그 토큰을 실어 보내는 구조입니다. 토큰 유효시간은 60분이며, 이 Provider가 만료 5분 전과 `401` 응답 시 자동으로 재발급합니다.

> 참고 문서: <https://cloud.kt.com/docs/open-api-guide/d/guide/how-to-use>

## Example Usage

```terraform
terraform {
  required_providers {
    ktcloud = {
      source  = "7-Victory/ktcloud"
      version = "~> 0.1"
    }
  }
}

provider "ktcloud" {
  zone = "d1" # 또는 환경변수 KTCLOUD_ZONE
}
```

자격증명은 절대 HCL에 하드코딩하지 말고 환경변수를 사용하세요.

```shell
export KTCLOUD_USERNAME='포털ID'
export KTCLOUD_PASSWORD='비밀번호'
export KTCLOUD_ZONE='d1'
```

## Schema

### Optional

- `api_base` (String) API 기본 도메인. 기본값 `https://api.ucloudbiz.olleh.com`. 환경변수 `KTCLOUD_API_BASE`.
- `zone` (String) Zone 약어. `d1`(DX-M1), `d2`(DX-Central), `gd1`(DX-G), `gd4`(DX-G-YS). 환경변수 `KTCLOUD_ZONE`. 실질적으로는 필수값입니다 — 비어 있으면 인증 단계에서 에러가 납니다.
- `username` (String) kt cloud 포털 사용자 ID. 환경변수 `KTCLOUD_USERNAME`. 실질적으로는 필수값입니다.
- `password` (String, Sensitive) kt cloud 포털 비밀번호. 환경변수 `KTCLOUD_PASSWORD`. 실질적으로는 필수값입니다.
- `domain_id` (String) 인증 domain id. 기본값 `default`. 환경변수 `KTCLOUD_DOMAIN_ID`.
- `project_name` (String) 인증 scope의 project name. 미지정 시 `username`과 동일하게 사용합니다. 환경변수 `KTCLOUD_PROJECT_NAME`.
- `insecure` (Boolean) TLS 인증서 검증을 건너뜁니다. 테스트 용도로만 사용하세요. 환경변수 `KTCLOUD_INSECURE`.
- `use_project_id_path` (Boolean) `true`로 설정하면 (volume 서비스를 제외한) 요청 경로에 project_id를 넣습니다 (`/{zone}/server/{project_id}/servers`). volume 서비스는 이 옵션과 무관하게 project_id가 항상 강제로 들어갑니다. 규격서와 경로가 다를 때만 사용하세요. 환경변수 `KTCLOUD_USE_PROJECT_ID_PATH`.
- `request_timeout` (Number) 개별 HTTP 요청 타임아웃(초). 기본값 `60`.

## 지원 범위

| 종류 | 이름 | 설명 |
|---|---|---|
| Resource | [`ktcloud_server`](resources/server.md) | 가상 서버(VM) |
| Resource | [`ktcloud_keypair`](resources/keypair.md) | SSH 키페어 |
| Resource | [`ktcloud_volume`](resources/volume.md) | 블록 스토리지 볼륨 |
| Resource | [`ktcloud_volume_attachment`](resources/volume_attachment.md) | 볼륨 ↔ VM 연결 |
| Data Source | [`ktcloud_flavors`](data-sources/flavors.md) | VM 스펙 목록 |
| Data Source | [`ktcloud_images`](data-sources/images.md) | OS 이미지 목록 |

## 알아두어야 할 것

- `ktcloud_server.networks.uuid`에는 kt cloud 네트워크 관리 API(`/nsm/v1/network`) 응답의 `networkId`가 아니라 **`refId`** 값을 써야 합니다. `networkId`를 넣으면 `Network ... could not be found` 400 에러가 납니다.
- `ktcloud_server.availability_zone`을 생략하면 `"DX-M1"`(d1 zone 기준)으로 기본 설정됩니다. `d1`이 아닌 zone(`d2`/`gd1`/`gd4`)을 쓰신다면 반드시 직접 지정하세요.
- 개인키(`ktcloud_keypair.private_key`)는 **tfstate에 평문으로 저장**됩니다. state를 암호화된 원격 백엔드에 두거나, 운영 환경에서는 로컬에서 만든 공개키를 등록하는 방식을 권장합니다.
