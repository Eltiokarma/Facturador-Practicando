// Package sunatstatus mantiene un estado en memoria del estado de los
// endpoints de SUNAT. Un goroutine los pinga periódicamente; los handlers
// HTTP leen el estado y el frontend muestra un banner cuando está caído.
package sunatstatus

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Endpoint identifica qué servicio de SUNAT estamos chequeando.
type Endpoint string

const (
	BillBeta Endpoint = "bill_beta"
	BillProd Endpoint = "bill_prod"
)

var endpointURLs = map[Endpoint]string{
	BillBeta: "https://e-beta.sunat.gob.pe/ol-ti-itcpfegem-beta/billService?wsdl",
	BillProd: "https://e-factura.sunat.gob.pe/ol-ti-itcpfegem/billService?wsdl",
}

type Status struct {
	Endpoint     Endpoint  `json:"endpoint"`
	Up           bool      `json:"up"`
	LastError    string    `json:"last_error,omitempty"`
	LastCheckAt  time.Time `json:"last_check_at"`
	ChangedAt    time.Time `json:"changed_at"`
}

type Checker struct {
	mu       sync.RWMutex
	statuses map[Endpoint]*Status
	client   *http.Client
	interval time.Duration
	logger   *slog.Logger
}

func New(interval time.Duration, logger *slog.Logger) *Checker {
	if interval <= 0 {
		interval = 2 * time.Minute
	}
	c := &Checker{
		statuses: make(map[Endpoint]*Status),
		client:   &http.Client{Timeout: 10 * time.Second},
		interval: interval,
		logger:   logger,
	}
	now := time.Now()
	// Estado inicial: optimista (asumimos up hasta que el primer ping diga lo contrario).
	for ep := range endpointURLs {
		c.statuses[ep] = &Status{Endpoint: ep, Up: true, LastCheckAt: now, ChangedAt: now}
	}
	return c
}

// Run arranca el loop de chequeos. Bloqueante; llamarlo en una goroutine.
func (c *Checker) Run(ctx context.Context) {
	// Primer chequeo inmediato (en background)
	go c.checkAll(ctx)
	t := time.NewTicker(c.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.checkAll(ctx)
		}
	}
}

func (c *Checker) checkAll(ctx context.Context) {
	for ep, url := range endpointURLs {
		c.check(ctx, ep, url)
	}
}

func (c *Checker) check(ctx context.Context, ep Endpoint, url string) {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, url, nil)
	if err != nil {
		c.update(ep, false, err.Error())
		return
	}
	resp, err := c.client.Do(req)
	if err != nil {
		c.update(ep, false, err.Error())
		return
	}
	defer resp.Body.Close()
	// 2xx, 3xx, 4xx → el server está vivo (4xx puede ser método no permitido,
	// pero el endpoint contesta).
	// 5xx → caído.
	if resp.StatusCode >= 500 {
		c.update(ep, false, fmt.Sprintf("HTTP %d", resp.StatusCode))
		return
	}
	c.update(ep, true, "")
}

func (c *Checker) update(ep Endpoint, up bool, lastErr string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	s, ok := c.statuses[ep]
	if !ok {
		s = &Status{Endpoint: ep, ChangedAt: now}
		c.statuses[ep] = s
	}
	if s.Up != up {
		s.ChangedAt = now
		if up {
			c.logger.Info("sunatstatus: endpoint recuperado", "endpoint", ep)
		} else {
			c.logger.Warn("sunatstatus: endpoint caído", "endpoint", ep, "err", lastErr)
		}
	}
	s.Up = up
	s.LastError = lastErr
	s.LastCheckAt = now
}

// Get devuelve un snapshot del estado de un endpoint.
func (c *Checker) Get(ep Endpoint) Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if s, ok := c.statuses[ep]; ok {
		return *s
	}
	return Status{Endpoint: ep}
}

// All devuelve un snapshot de todos los estados.
func (c *Checker) All() map[Endpoint]Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[Endpoint]Status, len(c.statuses))
	for k, v := range c.statuses {
		out[k] = *v
	}
	return out
}
