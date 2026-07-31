package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Volume 은 블록 스토리지 볼륨입니다.
type Volume struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Size        int64  `json:"size"`
	Status      string `json:"status"`
	VolumeType  string `json:"volume_type"`
	Bootable    string `json:"bootable"`
	CreatedAt   string `json:"created_at"`
	Zone        string `json:"availability_zone"`
	Attachments []struct {
		ServerID string `json:"server_id"`
		Device   string `json:"device"`
		ID       string `json:"id"`
	} `json:"attachments"`
}

// CreateVolumeOpts 는 볼륨 생성 파라미터입니다.
type CreateVolumeOpts struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Size        int64  `json:"size"`
	VolumeType  string `json:"volume_type,omitempty"`
	Zone        string `json:"availability_zone,omitempty"`
	SnapshotID  string `json:"snapshot_id,omitempty"`
	ImageRef    string `json:"imageRef,omitempty"`
}

type volumeWrapper struct {
	Volume Volume `json:"volume"`
}

type createVolumeRequest struct {
	Volume CreateVolumeOpts `json:"volume"`
}

// CreateVolume 은 볼륨을 생성합니다.
func (c *Client) CreateVolume(ctx context.Context, opts CreateVolumeOpts) (*Volume, error) {
	var out volumeWrapper
	if err := c.Do(ctx, http.MethodPost, ServiceVolume, PathVolumes, createVolumeRequest{Volume: opts}, &out); err != nil {
		return nil, err
	}
	if out.Volume.ID == "" {
		return nil, fmt.Errorf("볼륨 생성 응답에 volume.id 가 없습니다")
	}
	return &out.Volume, nil
}

// GetVolume 은 볼륨을 조회합니다.
func (c *Client) GetVolume(ctx context.Context, id string) (*Volume, error) {
	var out volumeWrapper
	if err := c.Do(ctx, http.MethodGet, ServiceVolume, fmt.Sprintf(PathVolumeFmt, id), nil, &out); err != nil {
		return nil, err
	}
	return &out.Volume, nil
}

// UpdateVolume 은 이름/설명을 수정합니다.
func (c *Client) UpdateVolume(ctx context.Context, id, name, description string) error {
	body := map[string]any{"volume": map[string]string{
		"name":        name,
		"description": description,
	}}
	return c.Do(ctx, http.MethodPut, ServiceVolume, fmt.Sprintf(PathVolumeFmt, id), body, nil)
}

// ExtendVolume 은 볼륨 크기를 늘립니다. (축소는 불가)
func (c *Client) ExtendVolume(ctx context.Context, id string, newSize int64) error {
	body := map[string]any{"os-extend": map[string]int64{"new_size": newSize}}
	return c.Do(ctx, http.MethodPost, ServiceVolume, fmt.Sprintf(PathVolumeActFmt, id), body, nil)
}

// DeleteVolume 은 볼륨을 삭제합니다.
func (c *Client) DeleteVolume(ctx context.Context, id string) error {
	err := c.Do(ctx, http.MethodDelete, ServiceVolume, fmt.Sprintf(PathVolumeFmt, id), nil, nil)
	if err != nil && IsNotFound(err) {
		return nil
	}
	return err
}

// WaitForVolumeStatus 는 볼륨이 원하는 상태가 될 때까지 폴링합니다.
func (c *Client) WaitForVolumeStatus(ctx context.Context, id string, targets []string, timeout time.Duration) (*Volume, error) {
	const interval = 4 * time.Second
	deadline := time.Now().Add(timeout)

	for {
		vol, err := c.GetVolume(ctx, id)
		if err != nil {
			return nil, err
		}
		cur := strings.ToLower(vol.Status)

		for _, t := range targets {
			if cur == strings.ToLower(t) {
				return vol, nil
			}
		}
		if cur == "error" || cur == "error_deleting" {
			return vol, fmt.Errorf("볼륨(%s) 이 %s 상태입니다", id, vol.Status)
		}
		if time.Now().After(deadline) {
			return vol, fmt.Errorf("볼륨(%s) 이 %v 상태가 되기를 %s 동안 기다렸으나 현재 상태는 %s 입니다",
				id, targets, timeout, vol.Status)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
}

// ---------------------------------------------------------------------------
// Volume Attachment (VM <-> Volume 연결)
// ---------------------------------------------------------------------------

// VolumeAttachment 는 VM 에 붙은 볼륨 연결 정보입니다.
type VolumeAttachment struct {
	ID       string `json:"id"`
	ServerID string `json:"serverId"`
	VolumeID string `json:"volumeId"`
	Device   string `json:"device"`
}

type attachmentWrapper struct {
	VolumeAttachment VolumeAttachment `json:"volumeAttachment"`
}

// AttachVolume 은 볼륨을 VM 에 연결합니다. device 는 비워두면 자동 할당입니다.
func (c *Client) AttachVolume(ctx context.Context, serverID, volumeID, device string) (*VolumeAttachment, error) {
	inner := map[string]string{"volumeId": volumeID}
	if device != "" {
		inner["device"] = device
	}
	body := map[string]any{"volumeAttachment": inner}

	var out attachmentWrapper
	path := fmt.Sprintf(PathVolAttachFmt, serverID)
	if err := c.Do(ctx, http.MethodPost, ServiceServer, path, body, &out); err != nil {
		return nil, err
	}
	return &out.VolumeAttachment, nil
}

// GetVolumeAttachment 는 연결 정보를 조회합니다.
func (c *Client) GetVolumeAttachment(ctx context.Context, serverID, volumeID string) (*VolumeAttachment, error) {
	var out attachmentWrapper
	path := fmt.Sprintf(PathVolAttachOne, serverID, volumeID)
	if err := c.Do(ctx, http.MethodGet, ServiceServer, path, nil, &out); err != nil {
		return nil, err
	}
	return &out.VolumeAttachment, nil
}

// DetachVolume 은 연결을 해제합니다.
func (c *Client) DetachVolume(ctx context.Context, serverID, volumeID string) error {
	path := fmt.Sprintf(PathVolAttachOne, serverID, volumeID)
	err := c.Do(ctx, http.MethodDelete, ServiceServer, path, nil, nil)
	if err != nil && IsNotFound(err) {
		return nil
	}
	return err
}
