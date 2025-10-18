# Intel GPU Prometheus Exporter

A Prometheus exporter for Intel GPU metrics using `intel_gpu_top`. This exporter provides detailed metrics about Intel integrated GPU usage, including engine utilization, memory bandwidth, power consumption, and per-process GPU usage.

## Features

- **GPU Engine Metrics**: Monitor Blitter, Render/3D, Video, and VideoEnhance engines
  - Busy, sema, and wait percentages per engine
- **Frequency Metrics**: Actual and requested GPU frequencies
- **Memory Bandwidth**: IMC read/write bandwidth
- **Power Metrics**: GPU and package power consumption
- **RC6 State**: GPU idle state percentage
- **Per-Process Metrics**: Track GPU memory usage for individual processes
  - Total, shared, resident, purgeable, and active memory
  - Per-process engine utilization (when available)

## Requirements

- Intel GPU with `intel_gpu_top` support
- Docker with `--privileged` access or appropriate capabilities
- Host PID namespace access (`--pid host`)
- Access to `/dev/dri` devices

## Quick Start

### Docker Compose

```yaml
services:
  intel-gpu-exporter:
    image: ghcr.io/mike1808/igpu-exporter:latest
    container_name: intel-gpu-exporter
    restart: unless-stopped
    privileged: true
    pid: host
    environment:
      - REFRESH_PERIOD_MS=1000
      - PORT=8080
    volumes:
      - /dev/dri/:/dev/dri/
    ports:
      - "8080:8080"
    networks:
      - monitoring

networks:
  monitoring:
    external: true
```

### Docker Run

```bash
docker run -d \
  --name intel-gpu-exporter \
  --restart unless-stopped \
  --privileged \
  --pid host \
  -e REFRESH_PERIOD_MS=1000 \
  -v /dev/dri/:/dev/dri/ \
  -p 8080:8080 \
  ghcr.io/mike1808/igpu-exporter:latest
```

### Direct Binary

```bash
# Build
go build -o igpu-exporter

# Run
./igpu-exporter
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port for metrics endpoint | `8080` |
| `REFRESH_PERIOD_MS` | Metrics refresh period in milliseconds | `10000` |
| `DEVICE` | Specific Intel GPU device to monitor | (auto-detect) |
| `DEBUG` | Enable debug logging (`true` or `1`) | `false` |

## Metrics

All metrics are prefixed with `igpu_`.

### Engine Metrics

- `igpu_engines_busy_percent{engine="..."}` - Engine busy utilization %
- `igpu_engines_sema_percent{engine="..."}` - Engine semaphore utilization %
- `igpu_engines_wait_percent{engine="..."}` - Engine wait utilization %

Available engines: `Blitter/0`, `Render/3D/0`, `Video/0`, `VideoEnhance/0`

### System Metrics

- `igpu_frequency_actual` - Actual GPU frequency (MHz)
- `igpu_frequency_requested` - Requested GPU frequency (MHz)
- `igpu_imc_bandwidth_reads` - IMC memory reads (MiB/s)
- `igpu_imc_bandwidth_writes` - IMC memory writes (MiB/s)
- `igpu_interrupts` - Interrupts per second
- `igpu_period` - Sample period (ms)
- `igpu_power_gpu` - GPU power consumption (W)
- `igpu_power_package` - Package power consumption (W)
- `igpu_rc6` - RC6 residency %

### Per-Process Metrics

- `igpu_client_memory_total_bytes{pid="...", name="..."}` - Total memory
- `igpu_client_memory_shared_bytes{pid="...", name="..."}` - Shared memory
- `igpu_client_memory_resident_bytes{pid="...", name="..."}` - Resident memory
- `igpu_client_memory_purgeable_bytes{pid="...", name="..."}` - Purgeable memory
- `igpu_client_memory_active_bytes{pid="...", name="..."}` - Active memory
- `igpu_client_engine_busy_percent{pid="...", name="...", engine="..."}` - Per-process engine utilization

## Prometheus Configuration

Add to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'intel-gpu'
    static_configs:
      - targets: ['intel-gpu-exporter:8080']
```

## Example Queries

```promql
# Total GPU power consumption
igpu_power_gpu

# Video engine utilization
igpu_engines_busy_percent{engine="Video/0"}

# Memory used by ffmpeg
igpu_client_memory_resident_bytes{name="ffmpeg"}

# Top 5 processes by GPU memory usage
topk(5, igpu_client_memory_resident_bytes)
```

## Troubleshooting

### No client metrics appearing

If per-process metrics aren't showing up:

1. Ensure the container has `--privileged` and `--pid host`
2. Verify `intel_gpu_top` shows clients when run directly on the host
3. Check that `/dev/dri` is properly mounted
4. Some kernel/driver versions may not support per-process tracking

### Permission errors

If you see permission errors:

```bash
# On host, ensure debugfs is mounted
sudo mount -t debugfs none /sys/kernel/debug

# Check perf_event_paranoid setting
cat /proc/sys/kernel/perf_event_paranoid
# Set to -1 for full access (if needed)
sudo sysctl kernel.perf_event_paranoid=-1
```

## Development

### Building from source

```bash
# Clone the repository
git clone https://github.com/mike1808/igpu-exporter.git
cd igpu-exporter

# Build
go build -o igpu-exporter

# Run
./igpu-exporter
```

### Building Docker image

```bash
docker build -t igpu-exporter .
```

## License

MIT License - see LICENSE file for details

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.
