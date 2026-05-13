-- GRE (Guía de Remisión Electrónica) + credenciales API SUNAT.
--
-- A diferencia de facturas/boletas, la GRE no usa SOAP con Clave SOL.
-- SUNAT exige autenticación OAuth2 contra un endpoint distinto:
--   https://api-cpe.sunat.gob.pe/v1/contribuyente/gem
-- y credenciales que se obtienen como "Cliente API" desde Clave SOL.
--
-- Por eso agregamos columnas dedicadas en tenants. En modo DEMO no se
-- requieren — se simulan las respuestas.

ALTER TABLE tenants
    ADD COLUMN gre_client_id_cifrado     BYTEA,
    ADD COLUMN gre_client_secret_cifrado BYTEA;

-- Los campos específicos de GRE viven en payload JSONB del comprobante.
-- Reusamos la tabla comprobantes con tipo_documento='09' (Guía de Remisión Remitente).
-- Para que las consultas por estado y fecha sigan siendo rápidas, no
-- hace falta nada nuevo a nivel de tabla.

-- Tipo de documento de la guía: agregar 09 al CHECK de receptor_tipo_doc
-- ya no aplica porque los receptores siguen siendo del catálogo 06 (DNI/RUC/etc.).
-- El campo tipo_documento de comprobantes acepta cualquier valor del catálogo 01.
