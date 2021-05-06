package export

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/cfunkhouser/kasa"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	version = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "version",
		Help: "Version information about this binary",
		ConstLabels: map[string]string{
			"version": kasa.Version,
		},
	})

	httpRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Count of all HTTP requests",
	}, []string{"code", "method"})

	httpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds",
		Help: "Duration of all HTTP requests",
	}, []string{"code", "handler", "method"})
)

type deviceExporter struct {
	onTime     prometheus.Gauge
	relayState prometheus.Gauge
	rssi       prometheus.Gauge
	info       *prometheus.GaugeVec

	registry *prometheus.Registry
}

func newDeviceExporter() (*deviceExporter, error) {
	de := &deviceExporter{
		onTime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "kasa_on_time",
				Help: "Amount of time a Kasa device has been on.",
			},
		),
		relayState: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "kasa_relay_state",
				Help: "State of the relay for a given Kasa device.",
			},
		),
		rssi: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "kasa_rssi",
				Help: "RSSI of the Kasa device radio.",
			},
		),
		info: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kasa_device_info",
				Help: "Information describing the Kasa device.",
			},
			[]string{"alias", "id", "name", "model", "sw"},
		),

		registry: prometheus.NewRegistry(),
	}

	if err := de.registry.Register(de.onTime); err != nil {
		return nil, err
	}
	if err := de.registry.Register(de.relayState); err != nil {
		return nil, err
	}
	if err := de.registry.Register(de.rssi); err != nil {
		return nil, err
	}
	if err := de.registry.Register(de.info); err != nil {
		return nil, err
	}
	return de, nil
}

type Handler struct {
	sync.RWMutex
	exporters map[string]*deviceExporter
}

func (h *Handler) exporterFor(t string) (*deviceExporter, error) {
	h.RLock()
	de, has := h.exporters[t]
	h.RUnlock()

	if !has {
		var err error
		de, err = newDeviceExporter()
		if err != nil {
			return nil, err
		}

		h.Lock()
		h.exporters[t] = de
		h.Unlock()
	}
	return de, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	if target == "" {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Target Not Found")
		return
	}

	de, err := h.exporterFor(target)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Hrm, that ain't right: %v", err)
		return
	}

	promhttp.HandlerFor(de.registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
}

func New() *Handler {
	return &Handler{
		exporters: make(map[string]*deviceExporter),
	}
}
