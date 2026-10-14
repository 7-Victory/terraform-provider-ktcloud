package provider

import "strings"

// defaultVolumeType 은 volume_type/root_volume_type 을 생략했을 때 쓰는 기본값입니다.
const defaultVolumeType = "HDD-2000iops"

// resolveVolumeType 은 "SSD"/"HDD" 축약형을 kt cloud 가 실제로 쓰는 구체적인 타입
// 이름("SSD-2000iops"/"HDD-2000iops")으로 치환해 API 호출 직전에만 적용합니다.
// Terraform state/plan 에는 사용자가 적은 값을 그대로 두어야 하므로(plan modifier 로
// 값을 바꾸면 "Provider produced invalid plan" 에러가 남), 이 변환은 Create() 호출
// 지점에서만 수행합니다. 그 외 값(예: "SSD-9600iops", "HDD-QOS")은 그대로 둡니다.
func resolveVolumeType(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "", "HDD":
		return defaultVolumeType
	case "SSD":
		return "SSD-2000iops"
	default:
		return raw
	}
}
