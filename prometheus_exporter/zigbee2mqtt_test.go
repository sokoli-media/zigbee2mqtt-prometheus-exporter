package prometheus_exporter

import (
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
	"testing"
)

func getGaugeVecValue(t *testing.T, metric *prometheus.GaugeVec, labels prometheus.Labels) float64 {
	var m = &dto.Metric{}
	if err := metric.With(labels).Write(m); err != nil {
		t.Fatalf("couldnt get metric with metricsLabelsNames: %s", err)
	}
	return m.Gauge.GetValue()
}

func TestLoadIkeaTradfriPowerMeter(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	topic := "zigbee2mqtt/szafa rack"
	payload := "{\"current\":0.54,\"energy\":1.05,\"identify\":null,\"linkquality\":255,\"power\":102,\"power_on_behavior\":\"on\",\"state\":\"ON\",\"update\":{\"installed_version\":33816645,\"latest_version\":33816645,\"state\":\"idle\"},\"voltage\":239.1}"

	err := processMosquittoMessage(logger, topic, payload)
	require.NoError(t, err)

	require.Equal(t, getGaugeVecValue(t, powerMeterCurrentMetric, prometheus.Labels{"device": "szafa rack"}), 0.54)
	require.Equal(t, getGaugeVecValue(t, powerMeterEnergyMetric, prometheus.Labels{"device": "szafa rack"}), 1.05)
	require.Equal(t, getGaugeVecValue(t, powerMeterPowerMetric, prometheus.Labels{"device": "szafa rack"}), 102.0)
	require.Equal(t, getGaugeVecValue(t, powerMeterVoltageMetric, prometheus.Labels{"device": "szafa rack"}), 239.1)
}
