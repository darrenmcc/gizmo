package main

import (
	"github.com/darrenmcc/gizmo/examples/servers/kit/api"
	"github.com/darrenmcc/gizmo/server/kit"
	"github.com/kelseyhightower/envconfig"
)

func main() {
	var cfg api.Config
	envconfig.Process("", &cfg)

	// runs the HTTP _AND_ gRPC servers
	err := kit.Run(api.New(cfg))
	if err != nil {
		panic("problems running service: " + err.Error())
	}
}
