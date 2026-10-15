package provider

import "strings"

// defaultVolumeType 은 volume_type/root_volume_type 을 생략했을 때 쓰는 기본 축약형입니다.
// (실제 IOPS 등급 문자열이 아니라 "HDD" 축약형 — resolveVolumeType 이 size 에 맞게
// 구체적인 등급으로 풀어줍니다.)
const defaultVolumeType = "HDD"

// resolveVolumeType 은 "SSD"/"HDD" 축약형을 kt cloud 가 실제로 쓰는 구체적인 IOPS 등급
// 이름으로 치환해 API 호출 직전에만 적용합니다. 등급은 볼륨 크기(size, GB)에 따라
// 달라집니다 (/home/cloud-user/terraform/volumeiops 공식 가이드 기준). Terraform
// state/plan 에는 사용자가 적은 값을 그대로 두어야 하므로(plan modifier 로 값을 바꾸면
// "Provider produced invalid plan" 에러가 남), 이 변환은 Create() 호출 지점에서만
// 수행합니다. "SSD"/"HDD" 외의 값(예: "SSD-9600iops", "HDD-QOS")은 그대로 둡니다.
func resolveVolumeType(raw string, sizeGB int64) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "", "HDD":
		return hddTierFor(sizeGB)
	case "SSD":
		return ssdTierFor(sizeGB)
	default:
		return raw
	}
}

// ssdTierFor 는 SSD Volume IOPS 표를 그대로 구현합니다.
//
//	100GB 이하: 2,000 / 200GB: 4,800 / 300~400GB: 6,400 / 500~600GB: 8,000 / 700GB 이상: 9,600
func ssdTierFor(sizeGB int64) string {
	switch {
	case sizeGB <= 100:
		return "SSD-2000iops"
	case sizeGB <= 200:
		return "SSD-4800iops"
	case sizeGB <= 400:
		return "SSD-6400iops"
	case sizeGB <= 600:
		return "SSD-8000iops"
	default:
		return "SSD-9600iops"
	}
}

// hddTierFor 는 HDD Volume IOPS 표를 그대로 구현합니다.
//
//	100GB 이하: 2,000 / 200GB: 4,800 / 300GB: 5,600 / 400GB 이상: 6,000
func hddTierFor(sizeGB int64) string {
	switch {
	case sizeGB <= 100:
		return "HDD-2000iops"
	case sizeGB <= 200:
		return "HDD-4800iops"
	case sizeGB <= 300:
		return "HDD-5600iops"
	default:
		return "HDD-6000iops"
	}
}
