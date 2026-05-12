-- Resúmenes diarios de boletas (RC).
--
-- SUNAT permite/exige reportar las boletas (tipo 03) y sus notas en un
-- "Resumen Diario" consolidado, en lugar de una a una. El plazo de envío
-- es 7 días calendario contados desde la fecha de emisión de las boletas.
--
-- El flujo es asíncrono en dos pasos:
--   1) Enviamos el resumen → SUNAT devuelve un TICKET.
--   2) Consultamos el ticket → SUNAT devuelve el CDR final con el veredicto.

CREATE TABLE resumenes_diarios (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- Identificador del resumen: RC-YYYYMMDD-N
    serie           CHAR(8) NOT NULL,           -- "RC-yyyymmdd"
    correlativo     INT NOT NULL,               -- N (autoincremental por día)
    fecha_referencia DATE NOT NULL,             -- el día de las boletas que estamos resumiendo
    fecha_emision   DATE NOT NULL,              -- el día en que estamos enviando
    -- Estados:
    --   pendiente  → recién creado, sin enviar
    --   enviando   → el worker lo está mandando a SUNAT
    --   consultando → SUNAT devolvió ticket; estamos polling el resultado
    --   aceptado   → ticket consultado, SUNAT aceptó
    --   aceptado_con_obs → aceptado con observaciones
    --   rechazado  → SUNAT rechazó
    --   error      → error transitorio, se reintenta
    estado          TEXT NOT NULL DEFAULT 'pendiente'
        CHECK (estado IN ('pendiente','enviando','consultando','aceptado','aceptado_con_obs','rechazado','error')),
    ticket          TEXT,                       -- ticket de SUNAT (cuando lo recibimos)
    sunat_codigo    TEXT,
    sunat_mensaje   TEXT,
    xml_enviado     BYTEA,
    cdr_zip         BYTEA,
    cantidad_comprobantes INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, serie, correlativo)
);
CREATE INDEX idx_resumenes_tenant_fecha ON resumenes_diarios(tenant_id, fecha_referencia DESC);

-- Vínculo N:1 comprobante → resumen. Cada boleta solo puede estar en UN
-- resumen aceptado (si lo está, se considera reportada).
CREATE TABLE resumen_items (
    resumen_id      UUID NOT NULL REFERENCES resumenes_diarios(id) ON DELETE CASCADE,
    comprobante_id  UUID NOT NULL REFERENCES comprobantes(id) ON DELETE RESTRICT,
    PRIMARY KEY (resumen_id, comprobante_id)
);
CREATE INDEX idx_resumen_items_comprobante ON resumen_items(comprobante_id);
