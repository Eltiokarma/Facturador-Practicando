-- Modo demo por tenant + soporte de notas (crédito/débito).
--
-- Modo demo: tenants que no quieren (o no pueden todavía) firmar y enviar
-- a SUNAT realmente. Las emisiones se simulan, sirven para explorar la UI
-- o para training. En la UI se ve un banner DEMO permanente.
--
-- Notas de crédito (tipo 07) y débito (tipo 08) referencian un
-- comprobante previo. La referencia se guarda en columnas dedicadas para
-- poder hacer queries (por ejemplo, "¿qué notas tiene esta factura?").

ALTER TABLE tenants
    ADD COLUMN demo_mode BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE comprobantes
    ADD COLUMN ref_tipo_doc      CHAR(2),
    ADD COLUMN ref_serie         CHAR(4),
    ADD COLUMN ref_correlativo   BIGINT,
    ADD COLUMN motivo_codigo     TEXT,           -- catálogo 09 o 10
    ADD COLUMN motivo_descripcion TEXT;

CREATE INDEX idx_comprobantes_ref ON comprobantes(tenant_id, ref_tipo_doc, ref_serie, ref_correlativo)
    WHERE ref_tipo_doc IS NOT NULL;
