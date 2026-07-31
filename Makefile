SHELL := /bin/bash

# --- 여기를 본인 값으로 바꾸세요 -------------------------------------------
NAMESPACE  ?= 7-Victory
NAME       ?= ktcloud
VERSION    ?= 0.1.0
HOSTNAME   ?= registry.terraform.io
# ---------------------------------------------------------------------------

BINARY  := terraform-provider-$(NAME)
OS_ARCH := $(shell go env GOOS)_$(shell go env GOARCH)
PLUGIN_DIR := $(HOME)/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(NAME)/$(VERSION)/$(OS_ARCH)

.PHONY: help
help:
	@echo "make build     - 바이너리 빌드 (./$(BINARY))"
	@echo "make install   - 로컬 플러그인 디렉터리에 설치"
	@echo "make fmt       - gofmt 적용"
	@echo "make vet       - go vet 실행"
	@echo "make test      - 단위 테스트"
	@echo "make testacc   - 실 인프라 acceptance 테스트 (실제 과금 발생!)"
	@echo "make tidy      - go mod tidy"
	@echo "make clean     - 산출물 삭제"

.PHONY: build
build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY) .

.PHONY: install
install: build
	mkdir -p $(PLUGIN_DIR)
	cp $(BINARY) $(PLUGIN_DIR)/$(BINARY)_v$(VERSION)
	@echo "설치 완료: $(PLUGIN_DIR)"

.PHONY: fmt
fmt:
	gofmt -w .

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test ./... -v -timeout 120s

.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v -timeout 120m

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: clean
clean:
	rm -f $(BINARY)
