package main

import (
	"github.com/darrenmcc/gizmo/config"
	"github.com/darrenmcc/gizmo/examples/pubsub/api-kafka-websocket-pubsub/service"
	"github.com/darrenmcc/gizmo/pubsub"
	"github.com/darrenmcc/gizmo/pubsub/kafka"
	"github.com/darrenmcc/gizmo/server"
)

func main() {
	var cfg struct {
		Server *server.Config
		Kafka  *kafka.Config
	}
	config.LoadJSONFile("./config.json", &cfg)

	// set the pubsub's Log to be the same as server's
	pubsub.Log = server.Log

	// in case we want to override the port or log location via CLI
	server.SetConfigOverrides(cfg.Server)

	server.Init("gamestream-example", cfg.Server)

	err := server.Register(service.NewStreamService(cfg.Server.HTTPPort, cfg.Kafka))
	if err != nil {
		server.Log.Fatal(err)
	}

	if err = server.Run(); err != nil {
		server.Log.Fatal(err)
	}
}
