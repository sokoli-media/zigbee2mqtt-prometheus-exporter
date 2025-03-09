package main

import (
	"log/slog"
	"os"
	"zigbee2mqtt-prometheus-exporter/prometheus_exporter"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	mosquittoConfig := prometheus_exporter.MosquittoConfig{
		Broker:   os.Getenv("MOSQUITTO_BROKER"),
		ClientId: os.Getenv("MOSQUITTO_CLIENT_ID"),
		Username: os.Getenv("MOSQUITTO_USERNAME"),
		Password: os.Getenv("MOSQUITTO_PASSWORD"),
	}

	if mosquittoConfig.Broker == "" || mosquittoConfig.ClientId == "" || mosquittoConfig.Username == "" || mosquittoConfig.Password == "" {
		logger.Error("one or more environment variables are not set")
		return
	}

	prometheus_exporter.RunHTTPServer(logger, mosquittoConfig)
}
