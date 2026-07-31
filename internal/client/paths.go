package client

// ---------------------------------------------------------------------------
// API 하위 경로 모음
//
// ★★★ 중요 ★★★
// kt cloud Open API 규격서(각 서비스별 페이지)와 경로가 다르면 **이 파일만** 고치면 됩니다.
// 리소스/데이터소스 코드는 전부 아래 상수를 참조합니다.
//
// 최종 URL = {api_base}/{zone}/{service}{path}
//   예) https://api.ucloudbiz.olleh.com/d1/server/flavors
// ---------------------------------------------------------------------------

const (
	// --- Virtual Machine (service: server) ---
	PathFlavors      = "/flavors/detail" // 상세 조회. 400/404 나면 "/flavors" 로 변경
	PathServers      = "/servers"
	PathServerFmt    = "/servers/%s"
	PathServerActFmt = "/servers/%s/action"
	PathServerDetail = "/servers/detail"
	PathVolAttachFmt = "/servers/%s/os-volume_attachments"
	PathVolAttachOne = "/servers/%s/os-volume_attachments/%s"

	// --- SSH Keypair (service: server) ---
	PathKeypairs   = "/os-keypairs"
	PathKeypairFmt = "/os-keypairs/%s"

	// --- Volume (service: volume) ---
	PathVolumes      = "/volumes"
	PathVolumeFmt    = "/volumes/%s"
	PathVolumeActFmt = "/volumes/%s/action"
	PathVolumeDetail = "/volumes/detail"

	// --- Image (service: image) ---
	PathImages   = "/v2/images"
	PathImageFmt = "/v2/images/%s"

	// --- Networking / Tier (service: nc) ---
	PathNetworks = "/networks"
	PathSubnets  = "/subnets"
)
