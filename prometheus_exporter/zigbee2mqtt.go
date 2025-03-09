package prometheus_exporter

import (
	"encoding/json"
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"log/slog"
	"regexp"
	"sync"
	"time"
)

var labels = []string{"device"}
var powerMeterCurrentMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{Name: "zigbee_power_meter_current"}, labels)
var powerMeterEnergyMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{Name: "zigbee_power_meter_energy_total"}, labels)
var powerMeterPowerMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{Name: "zigbee_power_meter_power"}, labels)
var powerMeterVoltageMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{Name: "zigbee_power_meter_voltage"}, labels)

var lastUpdateMetric = promauto.NewGauge(prometheus.GaugeOpts{Name: "zigbee_last_update"})
var unknownTopicMetric = promauto.NewCounterVec(prometheus.CounterOpts{Name: "zigbee_unknown_topic"}, []string{"topic"})

type IkeaTradfriPowerMeter struct {
	Current         float64 `json:"current"`
	Energy          float64 `json:"energy"`
	LinkQuality     int     `json:"linkquality"`
	Power           float64 `json:"power"`
	PowerOnBehavior string  `json:"power_on_behavior"`
	State           string  `json:"state"`
	Voltage         float64 `json:"voltage"`
}

func tryLoadingDeviceMetrics(logger *slog.Logger, deviceName string, payload string) {
	var ikeaTradfriPowerMeter IkeaTradfriPowerMeter
	err := json.Unmarshal([]byte(payload), &ikeaTradfriPowerMeter)
	if err == nil {
		metricLabels := prometheus.Labels{"device": deviceName}
		powerMeterCurrentMetric.With(metricLabels).Set(ikeaTradfriPowerMeter.Current)
		powerMeterEnergyMetric.With(metricLabels).Set(ikeaTradfriPowerMeter.Energy)
		powerMeterPowerMetric.With(metricLabels).Set(ikeaTradfriPowerMeter.Power)
		powerMeterVoltageMetric.With(metricLabels).Set(ikeaTradfriPowerMeter.Voltage)
		return
	}

	logger.Warn("couldn't find a matching schema for the specified payload")
}

func processMosquittoMessage(logger *slog.Logger, topic string, payload string) error {
	logger.Info("received message", "topic", topic, "payload", payload)

	if matches := regexp.MustCompile(`^zigbee2mqtt/([^/]+)$`).FindStringSubmatch(topic); len(matches) > 1 {
		deviceName := matches[1]
		tryLoadingDeviceMetrics(logger, deviceName, payload)
	} else {
		unknownTopicMetric.With(prometheus.Labels{"topic": topic}).Inc()
		logger.Warn(fmt.Sprintf("unknown topic: %s", topic))
	}

	lastUpdateMetric.SetToCurrentTime()
	return nil
}

func CollectZigbee2MQTTDevices(logger *slog.Logger, config MosquittoConfig, wg *sync.WaitGroup, quit chan bool) {
	opts := MQTT.NewClientOptions()
	opts.AddBroker(config.Broker)
	opts.SetClientID(config.ClientId)
	opts.SetUsername(config.Username)
	opts.SetPassword(config.Password)
	opts.SetCleanSession(false)

	choke := make(chan MQTT.Message)

	opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
		choke <- msg
	})

	logger.Info("connecting to mqtt")
	client := MQTT.NewClient(opts)

	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		logger.Error("couldn't connect to mqtt", "error", token.Error())
		return
	}

	logger.Info("subscribing to mqtt topics")
	token = client.Subscribe("zigbee2mqtt/#", byte(2), nil)
	if token.WaitTimeout(5*time.Second) && token.Error() != nil {
		logger.Error("couldn't subscribe to mqtt topics", "error", token.Error())
		return
	}

	logger.Info("waiting for zigbee2mqtt updates on mqtt")
	for {
		select {
		case message := <-choke:
			topic := message.Topic()
			payload := string(message.Payload())

			err := processMosquittoMessage(logger, topic, payload)
			if err != nil {
				logger.Error(
					"couldn't process mqtt message",
					"topic",
					message.Topic(),
					"payload",
					string(message.Payload()),
					"error",
					err,
				)
			}
		case <-quit:
			logger.Info("disconnecting from mqtt gracefully")
			client.Disconnect(250)
			wg.Done()
			return
		}
	}
}
