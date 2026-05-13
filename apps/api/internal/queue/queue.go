// Package queue encapsula la cola asíncrona de envíos a SUNAT, sobre Redis.
//
// Diseño:
//   - El cliente HTTP encola un job con (tenant_id, comprobante_id) y devuelve 202.
//   - El worker (en el mismo proceso, otra goroutine) consume el job: lee
//     el comprobante de DB, llama al motor PHP, actualiza el resultado.
//   - El estado canónico está en Postgres. Redis solo lleva la cola.
package queue

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

const (
	TypeEmit             = "comprobante:emit"
	TypeGuiaStatus       = "guia:status"
	TypeResumenEnviar    = "resumen:enviar"
	TypeResumenStatus    = "resumen:status"
	TypeAnularEnviar     = "anular:enviar"
	TypeAnularStatus     = "anular:status"
)

type EmitPayload struct {
	TenantID      uuid.UUID `json:"tenant_id"`
	ComprobanteID uuid.UUID `json:"comprobante_id"`
}

type ResumenPayload struct {
	TenantID  uuid.UUID `json:"tenant_id"`
	ResumenID uuid.UUID `json:"resumen_id"`
}

type AnularPayload struct {
	TenantID    uuid.UUID `json:"tenant_id"`
	AnulacionID uuid.UUID `json:"anulacion_id"`
}

type Client struct {
	c *asynq.Client
}

func NewClient(redisAddr string) *Client {
	return &Client{c: asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})}
}

func (c *Client) Close() error { return c.c.Close() }

// EnqueueEmit programa el envío de un comprobante. Reintentos automáticos
// con backoff exponencial (1s, 2s, 4s, 8s, 16s, 32s, 64s).
func (c *Client) EnqueueEmit(ctx context.Context, p EmitPayload) error {
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}
	task := asynq.NewTask(TypeEmit, body,
		asynq.MaxRetry(7),
		asynq.Timeout(2*time.Minute),
		asynq.Retention(48*time.Hour),
	)
	_, err = c.c.EnqueueContext(ctx, task)
	return err
}

// EnqueueResumenEnviar programa el envío del resumen diario.
func (c *Client) EnqueueResumenEnviar(ctx context.Context, p ResumenPayload) error {
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}
	task := asynq.NewTask(TypeResumenEnviar, body,
		asynq.MaxRetry(7),
		asynq.Timeout(2*time.Minute),
		asynq.Retention(7*24*time.Hour),
	)
	_, err = c.c.EnqueueContext(ctx, task)
	return err
}

// EnqueueResumenStatus programa la consulta del ticket de un resumen.
// delay difiere la ejecución (SUNAT tarda en procesar resúmenes).
func (c *Client) EnqueueResumenStatus(ctx context.Context, p ResumenPayload, delay time.Duration) error {
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}
	task := asynq.NewTask(TypeResumenStatus, body,
		asynq.MaxRetry(20),
		asynq.Timeout(1*time.Minute),
		asynq.Retention(7*24*time.Hour),
		asynq.ProcessIn(delay),
	)
	_, err = c.c.EnqueueContext(ctx, task)
	return err
}

// EnqueueGuiaStatus consulta el ticket de una GRE async.
func (c *Client) EnqueueGuiaStatus(ctx context.Context, p EmitPayload, delay time.Duration) error {
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}
	task := asynq.NewTask(TypeGuiaStatus, body,
		asynq.MaxRetry(20),
		asynq.Timeout(1*time.Minute),
		asynq.Retention(7*24*time.Hour),
		asynq.ProcessIn(delay),
	)
	_, err = c.c.EnqueueContext(ctx, task)
	return err
}

func (c *Client) EnqueueAnularEnviar(ctx context.Context, p AnularPayload) error {
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}
	task := asynq.NewTask(TypeAnularEnviar, body,
		asynq.MaxRetry(7), asynq.Timeout(2*time.Minute), asynq.Retention(7*24*time.Hour),
	)
	_, err = c.c.EnqueueContext(ctx, task)
	return err
}

func (c *Client) EnqueueAnularStatus(ctx context.Context, p AnularPayload, delay time.Duration) error {
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}
	task := asynq.NewTask(TypeAnularStatus, body,
		asynq.MaxRetry(20), asynq.Timeout(1*time.Minute),
		asynq.Retention(7*24*time.Hour), asynq.ProcessIn(delay),
	)
	_, err = c.c.EnqueueContext(ctx, task)
	return err
}
