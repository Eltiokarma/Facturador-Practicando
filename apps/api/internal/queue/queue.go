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

const TypeEmit = "comprobante:emit"

type EmitPayload struct {
	TenantID      uuid.UUID `json:"tenant_id"`
	ComprobanteID uuid.UUID `json:"comprobante_id"`
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
