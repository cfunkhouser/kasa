package export

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/cfunkhouser/kasa"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type deviceMetrics struct {
	onTime     prometheus.Gauge
	relayState prometheus.Gauge
	rssi       prometheus.Gauge
	info       *prometheus.GaugeVec
}

func (m *deviceMetrics) register(r prometheus.Registerer) error {
	if err := r.Register(m.onTime); err != nil {
		return err
	}
	if err := r.Register(m.relayState); err != nil {
		return err
	}
	if err := r.Register(m.rssi); err != nil {
		return err
	}
	if err := r.Register(m.info); err != nil {
		return err
	}
	return nil
}

type deviceExporter struct {
	daddr    *net.UDPAddr
	metrics  deviceMetrics
	registry *prometheus.Registry
}

var (
	ErrTooManyResponses = errors.New("got multiple responses for address")
	ErrNoDeviceResponse = errors.New("no response from device")
)

func (e *deviceExporter) update(ctx context.Context, laddr *net.UDPAddr) error {
	infos, err := kasa.GetSystemInformation(ctx, e.daddr, laddr, true)
	if err != nil {
		return err
	}
	if len(infos) > 1 {
		return fmt.Errorf("%w: %v", ErrTooManyResponses, e.daddr)
	}
	if len(infos) == 0 {
		return fmt.Errorf("%w: %v", ErrNoDeviceResponse, e.daddr)
	}
	info := infos[0]
	e.metrics.onTime.Set(float64(info.OnTime))
	e.metrics.relayState.Set(float64(info.RelayState))
	e.metrics.rssi.Set(float64(info.RSSI))
	e.metrics.info.With(prometheus.Labels{
		"alias": info.Alias,
		"id":    info.DeviceID,
		"name":  info.DevName,
		"model": info.Model,
		"sw":    info.SoftwareVersion,
	}).Set(1.0)
	return nil
}

func newDeviceExporter(daddr *net.UDPAddr) (*deviceExporter, error) {
	de := &deviceExporter{
		daddr: daddr,
		metrics: deviceMetrics{
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
		},

		registry: prometheus.NewRegistry(),
	}
	if err := de.metrics.register(de.registry); err != nil {
		return nil, err
	}
	return de, nil
}

type Handler struct {
	sync.RWMutex
	exporters map[string]*deviceExporter
	laddr     *net.UDPAddr
}

var ErrBadTarget = errors.New("bad target")

func (h *Handler) exporterFor(t string) (*deviceExporter, error) {
	daddr, err := kasa.ParseAddr(t)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadTarget, err)
	}
	h.RLock()
	de, has := h.exporters[t]
	h.RUnlock()

	if !has {
		var err error
		de, err = newDeviceExporter(daddr)
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
	if err := de.update(r.Context(), h.laddr); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed polling Kasa device: %v", err)
		return
	}
	promhttp.HandlerFor(de.registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
}

type Option func(*Handler)

func WithLocalAddr(laddr *net.UDPAddr) Option {
	return func(h *Handler) {
		h.laddr = laddr
	}
}

func New(opts ...Option) *Handler {
	h := &Handler{
		exporters: make(map[string]*deviceExporter),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}
