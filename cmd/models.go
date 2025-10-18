package main

import "github.com/prometheus/client_golang/prometheus"

type EngineMetrics struct {
	Busy float64 `json:"busy"`
	Sema float64 `json:"sema"`
	Wait float64 `json:"wait"`
	Unit string  `json:"unit"`
}

type FrequencyMetrics struct {
	Actual    float64 `json:"actual"`
	Requested float64 `json:"requested"`
}

type IMCBandwidthMetrics struct {
	Reads  float64 `json:"reads"`
	Writes float64 `json:"writes"`
}

type InterruptsMetrics struct {
	Count float64 `json:"count"`
}

type PeriodMetrics struct {
	Duration float64 `json:"duration"`
}

type PowerMetrics struct {
	GPU     float64 `json:"GPU"`
	Package float64 `json:"Package"`
}

type RC6Metrics struct {
	Value float64 `json:"value"`
}

type ClientMemory struct {
	Total     string `json:"total"`
	Shared    string `json:"shared"`
	Resident  string `json:"resident"`
	Purgeable string `json:"purgeable"`
	Active    string `json:"active"`
}

type EngineClassMetrics struct {
	Busy string `json:"busy"`
	Unit string `json:"unit"`
}

type Client struct {
	Name   string `json:"name"`
	PID    string `json:"pid"`
	Memory struct {
		System ClientMemory `json:"system"`
	} `json:"memory"`
	EngineClasses map[string]EngineClassMetrics `json:"engine-classes"`
}

type GPUTopData struct {
	Engines      map[string]EngineMetrics `json:"engines"`
	Frequency    FrequencyMetrics         `json:"frequency"`
	IMCBandwidth IMCBandwidthMetrics      `json:"imc-bandwidth"`
	Interrupts   InterruptsMetrics        `json:"interrupts"`
	Period       PeriodMetrics            `json:"period"`
	Power        PowerMetrics             `json:"power"`
	RC6          RC6Metrics               `json:"rc6"`
	Clients      map[string]Client        `json:"clients,omitempty"`
}

type Metrics struct {
	enginesBusy           *prometheus.GaugeVec
	enginesSema           *prometheus.GaugeVec
	enginesWait           *prometheus.GaugeVec
	frequencyActual       prometheus.Gauge
	frequencyRequested    prometheus.Gauge
	imcBandwidthReads     prometheus.Gauge
	imcBandwidthWrites    prometheus.Gauge
	interrupts            prometheus.Gauge
	period                prometheus.Gauge
	powerGPU              prometheus.Gauge
	powerPackage          prometheus.Gauge
	rc6                   prometheus.Gauge
	clientMemoryTotal     *prometheus.GaugeVec
	clientMemoryShared    *prometheus.GaugeVec
	clientMemoryResident  *prometheus.GaugeVec
	clientMemoryPurgeable *prometheus.GaugeVec
	clientMemoryActive    *prometheus.GaugeVec
	clientEngineBusy      *prometheus.GaugeVec
}
