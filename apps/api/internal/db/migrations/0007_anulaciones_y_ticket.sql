-- 0007: comunicaciones de baja (anulación de CPE ya aceptados) y ticket GRE.
--
-- Comunicación de baja:
--   SUNAT permite anular una factura, boleta, NC o ND ya aceptada dentro
--   de los 7 días siguientes a la emisión. Se envía un documento UBL
--   VoidedDocuments con la lista de comprobantes a anular y el motivo.
--   Igual que resumen diario: flow ticket → consulta async.
--
-- Para GRE:
--   El flow OAuth2 también devuelve ticket. Necesitamos persistirlo en
--   comprobantes para poder consultar el resultado de forma async.

ALTER TABLE comprobantes
    ADD COLUMN sunat_ticket TEXT,
    ADD COLUMN anulado      BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN anulado_at   TIMESTAMPTZ;

-- Permitir el nuevo estado 'consultando' para comprobantes que están
-- esperando que SUNAT termine de procesar un ticket (GRE).
ALTER TABLE comprobantes DROP CONSTRAINT IF EXISTS comprobantes_estado_check;
ALTER TABLE comprobantes ADD CONSTRAINT comprobantes_estado_check CHECK (
    estado IN ('pendiente','enviando','consultando','aceptado','aceptado_con_obs','rechazado','anulado','error')
);

-- Anulaciones (comunicación de baja).
CREATE TABLE anulaciones (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    serie           CHAR(8) NOT NULL,           -- "RA-yyyymmdd"
    correlativo     INT NOT NULL,
    fecha_referencia DATE NOT NULL,             -- día de emisión de los comprobantes a anular
    fecha_emision   DATE NOT NULL,              -- día de envío de la comunicación
    estado          TEXT NOT NULL DEFAULT 'pendiente'
        CHECK (estado IN ('pendiente','enviando','consultando','aceptado','aceptado_con_obs','rechazado','error')),
    ticket          TEXT,
    sunat_codigo    TEXT,
    sunat_mensaje   TEXT,
    xml_enviado     BYTEA,
    cdr_zip         BYTEA,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, serie, correlativo)
);
CREATE INDEX idx_anulaciones_tenant_fecha ON anulaciones(tenant_id, fecha_referencia DESC);

CREATE TABLE anulacion_items (
    anulacion_id   UUID NOT NULL REFERENCES anulaciones(id) ON DELETE CASCADE,
    comprobante_id UUID NOT NULL REFERENCES comprobantes(id) ON DELETE RESTRICT,
    motivo         TEXT NOT NULL,
    PRIMARY KEY (anulacion_id, comprobante_id)
);
CREATE INDEX idx_anulacion_items_comprobante ON anulacion_items(comprobante_id);
