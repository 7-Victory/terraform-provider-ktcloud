# Rocky Linux 작업 가이드

Rocky Linux 서버 한 대에서 **개발 환경 준비 → 빌드 → 로컬 테스트 → API 검증 →
git push → 배포**까지 전부 진행하는 절차입니다. 각 단계가 "무슨 단계인지"를 먼저 적고,
그 다음에 명령어를 씁니다.

문서 맨 아래 **[STEP 8] 내가 직접 수정해야 하는 파일** 절에 손봐야 할 곳이 정리돼 있습니다.

---

## [STEP 0] 개발 환경 준비 — *도구 설치 단계*

**무슨 단계?** Provider 는 Go 로 만든 실행 파일이고, Terraform 이 그걸 플러그인으로
띄웁니다. 따라서 서버에 Go(컴파일러), Terraform(테스트용), Git(버전관리)이 필요합니다.

```bash
sudo dnf install -y git make gcc tar
```

### Go 설치 (1.22 이상)

Rocky 기본 저장소 버전은 낮을 수 있으니 공식 tarball 을 쓰세요.

```bash
cd /tmp
curl -fsSLO https://go.dev/dl/go1.23.6.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.23.6.linux-amd64.tar.gz

# PATH 등록 (로그인 셸마다 적용되도록)
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc
source ~/.bashrc

go version   # go version go1.23.6 linux/amd64
```

### Terraform 설치

```bash
sudo dnf install -y dnf-plugins-core
sudo dnf config-manager --add-repo https://rpm.releases.hashicorp.com/RHEL/hashicorp.repo
sudo dnf install -y terraform

terraform version   # v1.5 이상이면 OK
```

> 사내 프록시 환경이라면 Go 모듈 다운로드용으로
> `go env -w GOPROXY=https://proxy.golang.org,direct` 또는 사내 프록시 주소를 설정하세요.

---

## [STEP 1] 프로젝트 배치 및 이름 바꾸기 — *스캐폴딩 단계*

**무슨 단계?** 코드에 박혀 있는 `7-Victory` 같은 자리표시자를 본인 GitHub 계정으로 바꿉니다.
Go 모듈 경로와 Terraform Registry 주소가 여기서 결정되므로 **가장 먼저** 해야 합니다.

```bash
mkdir -p ~/work && cd ~/work
# 받은 압축파일을 풀거나 파일을 복사해 둡니다
cd terraform-provider-ktcloud
```

`7-Victory` 를 본인 GitHub 계정/조직명으로 일괄 치환합니다 (예: `gildong`):

```bash
export GH_OWNER=gildong

grep -rl '7-Victory' --include='*.go' --include='*.tf' --include='Makefile' \
     --include='*.md' . | xargs sed -i "s|7-Victory|${GH_OWNER}|g"

grep -rn "${GH_OWNER}" go.mod main.go Makefile   # 치환 확인
```

바뀌는 것:

- `go.mod` 의 모듈 경로 → `github.com/gildong/terraform-provider-ktcloud`
- 모든 Go 파일의 import 경로
- `main.go` 의 `Address` → `registry.terraform.io/gildong/ktcloud`
- `Makefile` 의 `NAMESPACE`

---

## [STEP 2] 의존성 정리 및 빌드 — *컴파일 단계*

**무슨 단계?** Go 가 필요한 라이브러리를 내려받고(`go mod tidy`) 실행 파일을
만듭니다(`go build`). 여기서 에러가 없으면 문법·타입은 전부 정상입니다.

```bash
go mod tidy      # go.sum 생성 + 의존성 확정 (네트워크 필요)
go vet ./...     # 정적 검사
make build       # ./terraform-provider-ktcloud 생성
ls -lh terraform-provider-ktcloud
```

`go mod tidy` 가 네트워크 오류로 실패하면 프록시 설정 문제입니다:

```bash
go env -w GOPROXY=https://proxy.golang.org,direct GOSUMDB=sum.golang.org
```

---

## [STEP 3] Open API 실제 응답 확인 — *스펙 검증 단계* ⚠️ 중요

**무슨 단계?** 이 Provider 는 kt cloud 가 공개한 "OpenStack 호환" 규격에 맞춰 경로와
필드명을 작성했습니다. 하지만 서비스별 규격서에서 세부 경로/필드가 다를 수 있으므로,
**Terraform 을 돌리기 전에 curl 로 실제 응답을 한 번 확인**하세요. 이 단계를 건너뛰면
나중에 원인 찾기 어려운 404/400 에 시간을 씁니다.

### 3-1. 인증 토큰 발급

```bash
export KT_ID='포털ID'
export KT_PW='비밀번호'
export KT_ZONE='d1'      # d1 / d2 / gd1 / gd4

curl -s -D /tmp/hdr.txt -o /tmp/token.json \
  -X POST "https://api.ucloudbiz.olleh.com/${KT_ZONE}/identity/auth/tokens" \
  -H 'Content-Type: application/json' \
  -d "{
    \"auth\": {
      \"identity\": {
        \"methods\": [\"password\"],
        \"password\": { \"user\": {
          \"domain\": { \"id\": \"default\" },
          \"name\": \"${KT_ID}\",
          \"password\": \"${KT_PW}\"
        }}
      },
      \"scope\": { \"project\": {
        \"domain\": { \"id\": \"default\" },
        \"name\": \"${KT_ID}\"
      }}
    }
  }"

# 201 Created + X-Subject-Token 확인
grep -i -E 'HTTP/|x-subject-token' /tmp/hdr.txt

export KT_TOKEN=$(grep -i '^x-subject-token' /tmp/hdr.txt | tr -d '\r' | awk '{print $2}')
echo "${KT_TOKEN:0:20}..."
```

`token.project.id` 도 메모해 두세요 (콘솔 > Token 메뉴의 Project ID 와 동일):

```bash
python3 -c "import json;print(json.load(open('/tmp/token.json'))['token']['project']['id'])"
```

### 3-2. 각 엔드포인트 응답 확인

```bash
api() { curl -s -w '\n[HTTP %{http_code}]\n' \
  -H "X-Auth-Token: ${KT_TOKEN}" \
  "https://api.ucloudbiz.olleh.com/${KT_ZONE}$1"; }

api /server/flavors          # 문서에 나온 예제 경로 (200 이어야 정상)
api /server/flavors/detail   # Provider 기본값
api /server/servers
api /server/os-keypairs
api /volume/volumes
api /image/v2/images
api /nc/networks
```

**결과 판정**

| 결과 | 의미 | 조치 |
|---|---|---|
| `200` + JSON | 경로 정상 | 그대로 사용 |
| `404` | 경로가 다름 | `internal/client/paths.go` 수정 |
| `401` | 토큰 만료(60분) | 3-1 재실행 |
| `400`/`404` 인데 경로에 project id 필요 | 경로 형태가 다름 | `use_project_id_path = true` 시도 |

`/server/{project_id}/servers` 형태여야 한다면:

```bash
api "/server/$(cat /tmp/projid)/servers"
```

가 200 인지 확인하고, provider 블록에 `use_project_id_path = true` 를 넣으세요.

또한 응답 JSON 의 필드명이 코드와 다르면 `internal/client/*.go` 의 구조체
`json:"..."` 태그를 실제 응답에 맞게 고칩니다.

---

## [STEP 4] 로컬에서 Provider 테스트 — *개발 반복 단계*

**무슨 단계?** Terraform 은 보통 레지스트리에서 Provider 를 내려받지만, 개발 중에는
`dev_overrides` 로 로컬 바이너리를 직접 쓰게 만듭니다. 이러면 `terraform init` 없이
`go build` → `terraform plan` 만 반복할 수 있습니다.

### 4-1. `~/.terraformrc` 작성

```bash
cat > ~/.terraformrc <<EOF
provider_installation {
  dev_overrides {
    "${GH_OWNER}/ktcloud" = "$HOME/go/bin"
  }
  direct {}
}
EOF
```

바이너리를 그 경로에 설치:

```bash
mkdir -p ~/go/bin
go build -o ~/go/bin/terraform-provider-ktcloud .
```

### 4-2. 실행

```bash
cd examples/basic

export KTCLOUD_USERNAME="$KT_ID"
export KTCLOUD_PASSWORD="$KT_PW"
export KTCLOUD_ZONE="$KT_ZONE"

# dev_overrides 사용 시 terraform init 은 하지 않습니다 (경고만 나오면 정상)
terraform plan -var="network_id=<네트워크UUID>"
```

먼저 조회만 확인하는 게 안전합니다. `main.tf` 에서 리소스 부분을 주석 처리하고
data source 만 남긴 뒤:

```bash
terraform plan     # flavors / images 목록이 나오는지 확인
```

정상이면 리소스 주석을 풀고:

```bash
terraform apply -var="network_id=<네트워크UUID>"    # 실제 과금 발생
terraform show
terraform destroy -var="network_id=<네트워크UUID>"  # 반드시 정리
```

**디버깅**

```bash
TF_LOG=DEBUG terraform apply 2>&1 | tee /tmp/tf.log
grep -n 'kt cloud API 오류' /tmp/tf.log
```

### 4-3. 코드 수정 → 재시도 루프

```bash
go build -o ~/go/bin/terraform-provider-ktcloud . && terraform plan
```

---

## [STEP 5] Git 초기화 및 GitHub push — *버전관리 단계*

**무슨 단계?** 코드를 GitHub 저장소에 올립니다. Terraform Registry 배포는 GitHub 릴리스를
기준으로 동작하므로, 공개 배포까지 갈 거라면 GitHub 이 사실상 필수입니다.

### 5-1. SSH 키 준비 (HTTPS 대신 SSH 권장)

```bash
ssh-keygen -t ed25519 -C "you@example.com" -f ~/.ssh/id_ed25519
cat ~/.ssh/id_ed25519.pub
```

출력값을 GitHub → Settings → SSH and GPG keys → New SSH key 에 등록한 뒤:

```bash
ssh -T git@github.com     # "successfully authenticated" 확인
```

### 5-2. 커밋 전 점검 ⚠️

**비밀정보가 섞여 들어가지 않는지 반드시 확인하세요.**

```bash
git init -b main
git status                    # 올라갈 파일 목록 확인
git check-ignore -v *.tfvars .terraform 2>/dev/null   # .gitignore 동작 확인

# 혹시 모를 유출 점검
grep -rn --include='*.tf' --include='*.go' -iE 'password *= *"[^"]' . | grep -v 'Sensitive\|MarkdownDescription'
```

`.gitignore` 에 이미 `*.tfstate`, `*.tfvars`, `.env`, `*.pem` 등이 들어 있습니다.

### 5-3. 첫 커밋 & push

```bash
git config user.name  "Your Name"
git config user.email "you@example.com"

git add .
git commit -m "feat: kt cloud Terraform provider 초기 구현

- Keystone 토큰 인증 + 자동 갱신 클라이언트
- ktcloud_server / keypair / volume / volume_attachment 리소스
- ktcloud_flavors / images 데이터소스"

# GitHub 에서 빈 저장소(terraform-provider-ktcloud)를 먼저 만든 뒤
git remote add origin git@github.com:${GH_OWNER}/terraform-provider-ktcloud.git
git push -u origin main
```

### 5-4. 이후 작업 흐름

```bash
git switch -c feat/loadbalancer     # 브랜치 생성
# ... 코드 수정 ...
gofmt -w . && go vet ./... && make build
git add -A && git commit -m "feat: 로드밸런서 리소스 추가"
git push -u origin feat/loadbalancer
# GitHub 에서 Pull Request 생성 → 머지
git switch main && git pull
```

---

## [STEP 6] (선택) 사내 전용으로만 쓰기 — *사설 배포 단계*

**무슨 단계?** Terraform Registry 에 공개하지 않고 팀 내부에서만 쓰려면, 각 서버의
플러그인 디렉터리에 바이너리를 직접 넣으면 됩니다. 이 방식은 `dev_overrides` 없이
정식 `terraform init` 이 동작합니다.

```bash
make install
# → ~/.terraform.d/plugins/registry.terraform.io/<owner>/ktcloud/0.1.0/linux_amd64/
```

`~/.terraformrc` 의 `dev_overrides` 블록은 지우거나 주석 처리하세요
(둘이 동시에 있으면 override 가 우선합니다).

```bash
cd examples/basic
rm -rf .terraform .terraform.lock.hcl
terraform init      # 로컬 플러그인을 찾아 초기화
terraform plan
```

---

## [STEP 7] (선택) Terraform Registry 공개 배포 — *릴리스 단계*

**무슨 단계?** `terraform { required_providers { source = "owner/ktcloud" } }` 로 누구나
쓸 수 있게 공개합니다. Registry 는 GitHub 릴리스에 **GPG 로 서명된 체크섬**이 있을 것을
요구하므로, GPG 키와 goreleaser 가 필요합니다.

### 7-1. GPG 키 생성

```bash
sudo dnf install -y gnupg2
gpg --full-generate-key      # RSA 4096, 만료 없음 권장

gpg --list-secret-keys --keyid-format=long
export GPG_FINGERPRINT=<위에서 나온 지문(공백 없는 40자)>

gpg --armor --export "$GPG_FINGERPRINT" > /tmp/pubkey.asc
```

`/tmp/pubkey.asc` 내용을 Terraform Registry → Publish → GPG key 에 등록합니다.

### 7-2. GitHub Actions 시크릿 등록

GitHub 저장소 → Settings → Secrets and variables → Actions:

| 이름 | 값 |
|---|---|
| `GPG_PRIVATE_KEY` | `gpg --armor --export-secret-keys "$GPG_FINGERPRINT"` 출력 전체 |
| `PASSPHRASE` | GPG 키 암호 |

### 7-3. 릴리스 워크플로 추가

`.github/workflows/release.yml` 파일을 만듭니다:

```yaml
name: release
on:
  push:
    tags: ['v*']
permissions:
  contents: write
jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'
      - id: import_gpg
        uses: crazy-max/ghaction-import-gpg@v6
        with:
          gpg_private_key: ${{ secrets.GPG_PRIVATE_KEY }}
          passphrase: ${{ secrets.PASSPHRASE }}
      - uses: goreleaser/goreleaser-action@v6
        with:
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GPG_FINGERPRINT: ${{ steps.import_gpg.outputs.fingerprint }}
```

### 7-4. 태그를 밀어 릴리스

```bash
git add .github/workflows/release.yml
git commit -m "ci: goreleaser 릴리스 워크플로 추가"
git push

git tag v0.1.0
git push origin v0.1.0        # Actions 가 자동으로 릴리스 생성
```

로컬에서 직접 릴리스하려면:

```bash
go install github.com/goreleaser/goreleaser/v2@latest
export GITHUB_TOKEN=<personal access token>
goreleaser release --clean
```

### 7-5. Registry 에 등록

<https://registry.terraform.io> 로그인 → Publish → Provider → 저장소 선택.
저장소는 **public** 이어야 하고 이름이 `terraform-provider-ktcloud` 여야 합니다.

---

## [STEP 8] 내가 직접 수정해야 하는 파일

우선순위 순으로 정리했습니다. ★ 표시는 **반드시** 손봐야 하는 곳입니다.

### ★ 필수 — 이름/식별자

| 파일 | 위치 | 무엇을 |
|---|---|---|
| `go.mod` | 1행 `module` | `7-Victory` → 본인 GitHub 계정. STEP 1 의 `sed` 가 처리합니다. |
| `main.go` | `opts.Address` | `registry.terraform.io/7-Victory/ktcloud` → 본인 namespace |
| `Makefile` | `NAMESPACE`, `VERSION` | 본인 계정명, 릴리스할 버전 |
| 모든 `.go` 파일 | `import` 블록 | `github.com/7-Victory/...` 경로. `sed` 가 함께 처리 |
| `examples/basic/main.tf` | `source = "7-Victory/ktcloud"` | 본인 namespace |
| `~/.terraformrc` | `dev_overrides` 키 | `"<owner>/ktcloud"` 로 맞추기 |

### ★ 필수 — API 경로 검증 (STEP 3 결과 반영)

| 파일 | 무엇을 |
|---|---|
| **`internal/client/paths.go`** | **API 경로가 전부 여기 상수로 모여 있습니다.** curl 로 404 가 난 경로만 이 파일에서 고치면 리소스 코드는 건드릴 필요가 없습니다. 대표적으로 `PathFlavors` 를 `/flavors/detail` → `/flavors` 로 바꿔야 할 가능성이 있습니다. |

### 상황에 따라 — 응답 필드명

| 파일 | 구조체 | 무엇을 |
|---|---|---|
| `internal/client/server.go` | `Server`, `Address`, `CreateServerOpts` | curl 응답의 실제 키와 `json:"..."` 태그가 다르면 수정. 특히 공인 IP 판별에 쓰는 `OS-EXT-IPS:type` 이 응답에 없다면 `firstIPOfType` 로직도 함께 손보세요. |
| `internal/client/volume.go` | `Volume`, `CreateVolumeOpts` | 볼륨 타입 이름(`volume_type`)이 kt cloud 고유 값이면 문서 값으로 |
| `internal/client/misc.go` | `Flavor`, `Image`, `Keypair` | 조회 응답 래핑 키(`flavors` / `images` / `keypair`)가 다르면 수정 |
| `internal/client/client.go` | `Authenticate` | 인증 응답 파싱. 문서와 동일하게 작성했으니 보통 그대로 둡니다. |

### 선택 — 기능 확장

| 파일 | 무엇을 |
|---|---|
| `internal/provider/provider.go` | `Resources()` / `DataSources()` 에 새 리소스 등록. 여기에 추가하지 않으면 Terraform 이 인식하지 못합니다. |
| `internal/provider/server_resource.go` | `flavor_id` 를 재생성 대신 리사이즈로 바꾸려면 `RequiresReplace()` 를 제거하고 `Update` 에서 `ServerAction(ctx, id, map[string]any{"resize": map[string]string{"flavorRef": ...}})` 호출 + `confirmResize` 처리 추가 |
| `internal/provider/server_resource.go` | `serverCreateTimeout`, `serverDeleteTimeout` 값 조정 |
| 새 파일 | Load Balancer, NAS, 공인 IP, 포트포워딩 등. `internal/client/` 에 API 함수 → `internal/provider/` 에 리소스 → `provider.go` 에 등록, 3단 구조를 따르세요. |
| `.goreleaser.yml` | 배포 대상 OS/arch 조정 |
| `README.md` | 라이선스, 지원 범위 갱신 |

### 새 리소스를 추가할 때의 체크리스트

1. `internal/client/<서비스>.go` — 요청/응답 구조체 + CRUD 함수
2. `internal/client/paths.go` — 경로 상수 추가
3. `internal/provider/<이름>_resource.go` — `Metadata` / `Schema` / `Configure` / `Create` / `Read` / `Update` / `Delete` / `ImportState`
4. `internal/provider/provider.go` — `Resources()` 반환 목록에 팩토리 함수 추가
5. `examples/basic/main.tf` — 사용 예시
6. `README.md` — 인자 표

---

## 자주 만나는 오류

| 증상 | 원인 | 해결 |
|---|---|---|
| `kt cloud 인증 실패 ... HTTP 401` | ID/PW 오류, 또는 `project_name` 이 사용자 ID 와 다름 | `project_name` 을 명시 |
| `HTTP 404` (모든 요청) | 경로 규격 불일치 | STEP 3 재실행 → `paths.go` 수정 |
| `HTTP 400` (VM 생성) | 필수 파라미터 누락 (보통 네트워크) | `networks` 블록 확인 |
| `응답 파싱 실패` | 응답 래핑 키가 다름 | 오류 메시지에 찍힌 원본 JSON 을 보고 구조체 태그 수정 |
| `Provider ... not found` | `dev_overrides` 와 `source` 값 불일치 | 양쪽 namespace 를 동일하게 |
| plan 이 매번 재생성을 제안 | import 후 `networks`/`user_data` 가 state 에 없음 | HCL 에 실제 값 기입 또는 `ignore_changes` |
| 방화벽 때문에 API 호출 실패 | kt cloud VM 내부에서 실행 중 | Open API 엔드포인트로의 아웃바운드 오픈 필요 |
