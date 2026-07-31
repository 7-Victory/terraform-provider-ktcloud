// Terraform Provider for kt cloud (@D Platform / OpenStack 호환 Open API)
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/7-Victory/terraform-provider-ktcloud/internal/provider"
)

// goreleaser 빌드 시 -ldflags 로 주입됩니다.
var version = "dev"

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "디버거(delve) 연결용 모드로 실행")
	flag.Parse()

	opts := providerserver.ServeOpts{
		// registry.terraform.io/<NAMESPACE>/<TYPE>
		// NAMESPACE 를 본인 GitHub 계정/조직으로 바꾸세요.
		Address: "registry.terraform.io/7-Victory/ktcloud",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err.Error())
	}
}
