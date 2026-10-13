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

resource "ktcloud_server" "web" {
  name              = "tf-demo-web-03"
  flavor_id         = "" # data.ktcloud_flavors 로 조회한 id
  image_id          = "" # data.ktcloud_images 로 조회한 id
  keypair_name      = ktcloud_keypair.demo.name
  availability_zone = "DX-M1" # d1 zone 의 AZ 이름. 비워두면 500 에러가 날 수 있음
  root_volume_size  = 50

  networks {
    uuid = "" # data.ktcloud_networks 로 조회한 id (refId)
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

# resource "ktcloud_volume_attachment" "data" {
#   server_id = ktcloud_server.web.id
#   volume_id = ktcloud_volume.data.id
# }
# ↑ 주석 처리해서 destroy 되게 함 (볼륨 해제/detach 테스트)