terraform {
  required_version = ">= 1.5.0"

  required_providers {
    ktcloud = {
      # NAMESPACE 를 본인 것으로 바꾸세요.
      source  = "7-Victory/ktcloud"
      version = "~> 0.1"
    }
  }
}

# 자격증명은 코드에 넣지 말고 환경변수를 쓰세요.
#   export KTCLOUD_USERNAME='포털ID'
#   export KTCLOUD_PASSWORD='비밀번호'
#   export KTCLOUD_ZONE='d1'
provider "ktcloud" {
  # zone = "d1"
}

# ---------------------------------------------------------------------------
# 1) 스펙 / 이미지 조회
# ---------------------------------------------------------------------------
data "ktcloud_flavors" "small" {
  name_contains = "2x4" # 예: 2vCore 4GB
}

data "ktcloud_images" "rocky" {
  name_contains = "rocky"
}

output "available_flavors" {
  value = data.ktcloud_flavors.small.flavors
}

output "available_images" {
  value = data.ktcloud_images.rocky.images
}

# ---------------------------------------------------------------------------
# 2) SSH 키페어
# ---------------------------------------------------------------------------
resource "ktcloud_keypair" "demo" {
  name = "tf-demo-key"

  # 이미 가진 공개키를 등록하려면:
  # public_key = file("~/.ssh/id_rsa.pub")
  #
  # 생략하면 kt cloud 가 새 키를 만들고 private_key 에 담아줍니다.
}

output "private_key_pem" {
  value     = ktcloud_keypair.demo.private_key
  sensitive = true
}

# ---------------------------------------------------------------------------
# 3) VM 생성
# ---------------------------------------------------------------------------
variable "network_id" {
  description = "VM 을 붙일 네트워크(Tier) UUID"
  type        = string
}

resource "ktcloud_server" "web" {
  name             = "tf-demo-web-01"
  flavor_id        = data.ktcloud_flavors.small.flavors[0].id
  image_id         = data.ktcloud_images.rocky.images[0].id
  keypair_name     = ktcloud_keypair.demo.name
  root_volume_size = 50

  networks {
    uuid = var.network_id
  }

  user_data = <<-EOT
    #!/bin/bash
    dnf install -y nginx
    systemctl enable --now nginx
  EOT

  metadata = {
    managed_by = "terraform"
    env        = "demo"
  }
}

output "server_private_ip" {
  value = ktcloud_server.web.private_ip
}

# ---------------------------------------------------------------------------
# 4) 데이터 볼륨 추가 후 VM 에 연결
# ---------------------------------------------------------------------------
resource "ktcloud_volume" "data" {
  name = "tf-demo-data-01"
  size = 100
}

resource "ktcloud_volume_attachment" "data" {
  server_id = ktcloud_server.web.id
  volume_id = ktcloud_volume.data.id
}
