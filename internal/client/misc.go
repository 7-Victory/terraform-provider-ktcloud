package client

import (
	"context"
	"fmt"
	"net/http"
)

// ---------------------------------------------------------------------------
// SSH Keypair
// ---------------------------------------------------------------------------

// Keypair 는 SSH 키페어 정보입니다.
type Keypair struct {
	Name        string `json:"name"`
	PublicKey   string `json:"public_key"`
	PrivateKey  string `json:"private_key"` // 신규 생성 시에만 반환
	Fingerprint string `json:"fingerprint"`
}

type keypairWrapper struct {
	Keypair Keypair `json:"keypair"`
}

type createKeypairOpts struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key,omitempty"`
}

type createKeypairRequest struct {
	Keypair createKeypairOpts `json:"keypair"`
}

// CreateKeypair 는 키페어를 생성합니다. publicKey 가 빈 문자열이면 서버가 새 키를 만들고
// 응답의 private_key 로 개인키를 돌려줍니다.
func (c *Client) CreateKeypair(ctx context.Context, name, publicKey string) (*Keypair, error) {
	var out keypairWrapper
	body := createKeypairRequest{Keypair: createKeypairOpts{Name: name, PublicKey: publicKey}}
	if err := c.Do(ctx, http.MethodPost, ServiceServer, PathKeypairs, body, &out); err != nil {
		return nil, err
	}
	return &out.Keypair, nil
}

// GetKeypair 는 키페어를 조회합니다.
func (c *Client) GetKeypair(ctx context.Context, name string) (*Keypair, error) {
	var out keypairWrapper
	if err := c.Do(ctx, http.MethodGet, ServiceServer, fmt.Sprintf(PathKeypairFmt, name), nil, &out); err != nil {
		return nil, err
	}
	return &out.Keypair, nil
}

// DeleteKeypair 는 키페어를 삭제합니다.
func (c *Client) DeleteKeypair(ctx context.Context, name string) error {
	err := c.Do(ctx, http.MethodDelete, ServiceServer, fmt.Sprintf(PathKeypairFmt, name), nil, nil)
	if err != nil && IsNotFound(err) {
		return nil
	}
	return err
}

// ---------------------------------------------------------------------------
// Flavor (VM 스펙)
// ---------------------------------------------------------------------------

// Flavor 는 VM 스펙입니다.
type Flavor struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	VCPUs int64  `json:"vcpus"`
	RAM   int64  `json:"ram"`  // MB
	Disk  int64  `json:"disk"` // GB
}

type flavorsWrapper struct {
	Flavors []Flavor `json:"flavors"`
}

// ListFlavors 는 사용 가능한 스펙 목록을 조회합니다.
func (c *Client) ListFlavors(ctx context.Context) ([]Flavor, error) {
	var out flavorsWrapper
	if err := c.Do(ctx, http.MethodGet, ServiceServer, PathFlavors, nil, &out); err != nil {
		return nil, err
	}
	return out.Flavors, nil
}

// ---------------------------------------------------------------------------
// Image
// ---------------------------------------------------------------------------

// Image 는 OS 이미지 정보입니다.
type Image struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	DiskFmt  string `json:"disk_format"`
	MinDisk  int64  `json:"min_disk"`
	MinRAM   int64  `json:"min_ram"`
	Size     int64  `json:"size"`
	Created  string `json:"created_at"`
	OSDistro string `json:"os_distro"`
}

type imagesWrapper struct {
	Images []Image `json:"images"`
}

// ListImages 는 이미지 목록을 조회합니다.
func (c *Client) ListImages(ctx context.Context) ([]Image, error) {
	var out imagesWrapper
	if err := c.Do(ctx, http.MethodGet, ServiceImage, PathImages, nil, &out); err != nil {
		return nil, err
	}
	return out.Images, nil
}
