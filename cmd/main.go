package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func newMetrics() *Metrics {
	m := &Metrics{
		enginesBusy: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "igpu_engines_busy_percent",
			Help: "Engine busy utilisation %",
		}, []string{"engine"}),
		enginesSema: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "igpu_engines_sema_percent",
			Help: "Engine sema utilisation %",
		}, []string{"engine"}),
		enginesWait: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "igpu_engines_wait_percent",
			Help: "Engine wait utilisation %",
		}, []string{"engine"}),
		frequencyActual: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "igpu_frequency_actual",
			Help: "Frequency actual MHz",
		}),
		frequencyRequested: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "igpu_frequency_requested",
			Help: "Frequency requested MHz",
		}),
		imcBandwidthReads: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "igpu_imc_bandwidth_reads",
			Help: "IMC reads MiB/s",
		}),
		imcBandwidthWrites: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "igpu_imc_bandwidth_writes",
			Help: "IMC writes MiB/s",
		}),
		interrupts: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "igpu_interrupts",
			Help: "Interrupts/s",
		}),
		period: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "igpu_period",
			Help: "Period ms",
		}),
		powerGPU: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "igpu_power_gpu",
			Help: "GPU power W",
		}),
		powerPackage: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "igpu_power_package",
			Help: "Package power W",
		}),
		rc6: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "igpu_rc6",
			Help: "RC6 %",
		}),
		clientMemoryTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "igpu_client_memory_total_bytes",
			Help: "Client memory total bytes",
		}, []string{"pid", "name"}),
		clientMemoryShared: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "igpu_client_memory_shared_bytes",
			Help: "Client memory shared bytes",
		}, []string{"pid", "name"}),
		clientMemoryResident: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "igpu_client_memory_resident_bytes",
			Help: "Client memory resident bytes",
		}, []string{"pid", "name"}),
		clientMemoryPurgeable: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "igpu_client_memory_purgeable_bytes",
			Help: "Client memory purgeable bytes",
		}, []string{"pid", "name"}),
		clientMemoryActive: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "igpu_client_memory_active_bytes",
			Help: "Client memory active bytes",
		}, []string{"pid", "name"}),
		clientEngineBusy: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "igpu_client_engine_busy_percent",
			Help: "Client engine busy utilisation %",
		}, []string{"pid", "name", "engine"}),
	}

	prometheus.MustRegister(
		m.enginesBusy,
		m.enginesSema,
		m.enginesWait,
		m.frequencyActual,
		m.frequencyRequested,
		m.imcBandwidthReads,
		m.imcBandwidthWrites,
		m.interrupts,
		m.period,
		m.powerGPU,
		m.powerPackage,
		m.rc6,
		m.clientMemoryTotal,
		m.clientMemoryShared,
		m.clientMemoryResident,
		m.clientMemoryPurgeable,
		m.clientMemoryActive,
		m.clientEngineBusy,
	)

	return m
}

func (m *Metrics) update(data *GPUTopData) {
	// Reset client metrics to avoid stale data
	m.clientMemoryTotal.Reset()
	m.clientMemoryShared.Reset()
	m.clientMemoryResident.Reset()
	m.clientMemoryPurgeable.Reset()
	m.clientMemoryActive.Reset()
	m.clientEngineBusy.Reset()
	m.enginesBusy.Reset()
	m.enginesSema.Reset()
	m.enginesWait.Reset()

	for engineName, engine := range data.Engines {
		labels := prometheus.Labels{
			"engine": engineName,
		}
		m.enginesBusy.With(labels).Set(engine.Busy)
		m.enginesSema.With(labels).Set(engine.Sema)
		m.enginesWait.With(labels).Set(engine.Wait)
	}

	m.frequencyActual.Set(data.Frequency.Actual)
	m.frequencyRequested.Set(data.Frequency.Requested)

	m.imcBandwidthReads.Set(data.IMCBandwidth.Reads)
	m.imcBandwidthWrites.Set(data.IMCBandwidth.Writes)

	m.interrupts.Set(data.Interrupts.Count)

	m.period.Set(data.Period.Duration)

	m.powerGPU.Set(data.Power.GPU)
	m.powerPackage.Set(data.Power.Package)

	m.rc6.Set(data.RC6.Value)

	for _, client := range data.Clients {
		labels := prometheus.Labels{
			"pid":  client.PID,
			"name": client.Name,
		}

		if total, err := strconv.ParseFloat(client.Memory.System.Total, 64); err == nil {
			m.clientMemoryTotal.With(labels).Set(total)
		} else {
			log.Printf("Error parsing total memory for client %s: %v", client.Name, err)
		}
		if shared, err := strconv.ParseFloat(client.Memory.System.Shared, 64); err == nil {
			m.clientMemoryShared.With(labels).Set(shared)
		}
		if resident, err := strconv.ParseFloat(client.Memory.System.Resident, 64); err == nil {
			m.clientMemoryResident.With(labels).Set(resident)
		}
		if purgeable, err := strconv.ParseFloat(client.Memory.System.Purgeable, 64); err == nil {
			m.clientMemoryPurgeable.With(labels).Set(purgeable)
		}
		if active, err := strconv.ParseFloat(client.Memory.System.Active, 64); err == nil {
			m.clientMemoryActive.With(labels).Set(active)
		}

		// Update engine busy metrics
		for engineName, engineMetrics := range client.EngineClasses {
			if busy, err := strconv.ParseFloat(engineMetrics.Busy, 64); err == nil {
				engineLabels := prometheus.Labels{
					"pid":    client.PID,
					"name":   client.Name,
					"engine": engineName,
				}
				m.clientEngineBusy.With(engineLabels).Set(busy)
			} else {
				log.Printf("Error parsing busy for client %s engine %s: %v", client.Name, engineName, err)
			}
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string) bool {
	value := os.Getenv(key)
	return value == "true" || value == "1"
}

func main() {
	debug := getEnvBool("DEBUG")
	if !debug {
		log.SetFlags(log.LstdFlags)
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	metrics := newMetrics()

	port := getEnv("PORT", "8080")

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		addr := ":" + port
		log.Printf("Starting HTTP server on %s", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatal(err)
		}
	}()

	periodStr := getEnv("REFRESH_PERIOD_MS", "10000")
	period, err := strconv.Atoi(periodStr)
	if err != nil {
		log.Fatalf("Invalid REFRESH_PERIOD_MS: %v", err)
	}

	device := os.Getenv("DEVICE")
	var args []string
	args = append(args, "-J", "-s", strconv.Itoa(period))
	if device != "" {
		args = append(args, "-d", device)
	}

	cmd := exec.Command("intel_gpu_top", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Error creating stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Error starting intel_gpu_top: %v", err)
	}

	log.Printf("Started intel_gpu_top %v", args)

	done := make(chan struct{})

	// intel_gpu_top has a weird behavior when outputing in JSON mode
	// there are 3 behaviors that I have noticed:
	// 1. It outputs a regular JSON array like :
	//     [
	//     {
	//        // ...
	//     },
	//     {
	//        // ...
	//     },
	//     ]
	// 		This case can be parsed by using encoding/json#Decoder.
	// 2. It outputs a JSON array but with no commas like:
	//     [
	//     {
	//        // ...
	//     }
	//     {
	//        // ...
	//     }
	//     ]
	//     This cannot be parsed by regular Go methods
	// 3. It outputs just {}, {} and {} {} with no array.
	//
	// In order to support all cases we are just going to read byte by byte and trace brackets.
	go func() {
		defer close(done)

		reader := bufio.NewReader(stdout)
		var buffer bytes.Buffer
		braceDepth := 0
		inObject := false

		for {
			b, err := reader.ReadByte()
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Printf("Error reading: %v", err)
				break
			}

			// Track brace depth to detect complete objects
			if b == '{' {
				braceDepth++
				inObject = true
			}

			if inObject {
				buffer.WriteByte(b)
			}

			if b == '}' {
				braceDepth--
				if braceDepth == 0 && inObject {
					// We have a complete JSON object
					var data GPUTopData
					if err := json.Unmarshal(buffer.Bytes(), &data); err != nil {
						log.Printf("Error decoding JSON: %v", err)
					} else {
						if debug {
							log.Printf("Parsed data: %+v", data)
						}
						metrics.update(&data)
					}
					buffer.Reset()
					inObject = false
				}
			}
		}
	}()

	// Wait for either completion or signal
	select {
	case <-done:
		log.Println("Processing completed")
	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)
	}

	// Cleanup
	log.Println("Shutting down...")
	if err := cmd.Process.Kill(); err != nil {
		log.Printf("Error killing process: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		// Ignore error if we killed the process
		if _, ok := err.(*exec.ExitError); !ok {
			log.Printf("intel_gpu_top exited with error: %v", err)
		}
	}

	log.Println("Finished")
}
