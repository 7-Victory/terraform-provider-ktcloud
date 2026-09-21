---
page_title: "ktcloud_keypair Resource - terraform-provider-ktcloud"
subcategory: ""
description: |-
  kt cloud SSH 키페어를 관리합니다.
---

# ktcloud_keypair (Resource)

kt cloud SSH 키페어를 관리합니다.

## Example Usage

```terraform
resource "ktcloud_keypair" "demo" {
  name       = "tf-demo-key"
  public_key = file("~/.ssh/id_rsa.pub") # 생략하면 kt cloud가 새 키 생성
}

output "private_key_pem" {
  value     = ktcloud_keypair.demo.private_key
  sensitive = true
}
```

`public_key`를 생략하면 `private_key`(sensitive)에 개인키가 담깁니다. **개인키가 tfstate에 평문 저장되므로** state를 암호화된 원격 백엔드에 두거나, 운영 환경에서는 로컬에서 만든 공개키를 등록하는 방식을 권장합니다.

## Schema

### Required

- `name` (String) 키페어 이름. 변경 시 **재생성**.

### Optional

- `public_key` (String) 등록할 공개키(OpenSSH 형식). 생략하면 kt cloud가 새 키를 생성하고 `private_key`에 개인키가 담깁니다. 변경 시 **재생성**.

### Read-Only

- `id` (String) 키페어 이름과 동일합니다.
- `private_key` (String, Sensitive) 새로 생성된 경우에만 값이 있습니다. **state에 평문 저장되므로 state 파일 보안에 주의하세요.**
- `fingerprint` (String) 키 지문.

## Import

```shell
terraform import ktcloud_keypair.demo <키페어이름>
```
