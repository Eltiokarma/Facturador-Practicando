-- Migration inicial. Multi-tenant por columna tenant_id.
-- Toda tabla de negocio incluye tenant_id (uuid) y se indexa por (tenant_id, ...).

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ------------------------------------------------------------------------
-- Tenants (empresas / RUCs)
-- ------------------------------------------------------------------------
CREATE TABLE tenants (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ruc             CHAR(11) NOT NULL UNIQUE,
    razon_social    TEXT NOT NULL,
    nombre_comercial TEXT,
    direccion_fiscal TEXT,
    ubigeo          CHAR(6),
    sunat_mode      TEXT NOT NULL DEFAULT 'beta' CHECK (sunat_mode IN ('beta','prod')),
    -- Credenciales SUNAT (usuario secundario Clave SOL).
    -- El password se guarda cifrado con la passphrase del operador, no en plano.
    sunat_usuario_sol_cifrado BYTEA,
    sunat_clave_sol_cifrada   BYTEA,
    -- Referencia al archivo .p12 (path relativo dentro del volumen /app/certs).
    cert_path       TEXT,
    cert_pass_cifrado BYTEA,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ------------------------------------------------------------------------
-- Usuarios del sistema (multi-usuario por tenant)
-- ------------------------------------------------------------------------
CREATE TABLE users (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email        TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    nombre       TEXT NOT NULL,
    rol          TEXT NOT NULL CHECK (rol IN ('dueno','contador','cajero')),
    activo       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, email)
);
CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE UNIQUE INDEX idx_users_email_lower ON users(tenant_id, lower(email));

-- ------------------------------------------------------------------------
-- Series y correlativos (atómicos)
-- ------------------------------------------------------------------------
-- tipo_documento usa el catálogo SUNAT 01 (Factura), 03 (Boleta), 07 (NC), 08 (ND), etc.
CREATE TABLE series (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    tipo_documento  CHAR(2) NOT NULL,
    serie           CHAR(4) NOT NULL,
    correlativo     BIGINT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, tipo_documento, serie)
);

-- ------------------------------------------------------------------------
-- Comprobantes electrónicos (CPE)
-- ------------------------------------------------------------------------
CREATE TABLE comprobantes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    tipo_documento  CHAR(2) NOT NULL,
    serie           CHAR(4) NOT NULL,
    correlativo     BIGINT NOT NULL,
    fecha_emision   DATE NOT NULL,
    moneda          CHAR(3) NOT NULL DEFAULT 'PEN',
    -- Receptor
    receptor_tipo_doc CHAR(1) NOT NULL,
    receptor_num_doc TEXT NOT NULL,
    receptor_razon  TEXT NOT NULL,
    -- Totales (calculados server-side, nunca confiados al cliente)
    total_gravado   NUMERIC(12,2) NOT NULL DEFAULT 0,
    total_exonerado NUMERIC(12,2) NOT NULL DEFAULT 0,
    total_inafecto  NUMERIC(12,2) NOT NULL DEFAULT 0,
    total_gratuito  NUMERIC(12,2) NOT NULL DEFAULT 0,
    igv             NUMERIC(12,2) NOT NULL DEFAULT 0,
    isc             NUMERIC(12,2) NOT NULL DEFAULT 0,
    icbper          NUMERIC(12,2) NOT NULL DEFAULT 0,
    total           NUMERIC(12,2) NOT NULL DEFAULT 0,
    -- Estado del envío a SUNAT
    estado          TEXT NOT NULL DEFAULT 'pendiente'
        CHECK (estado IN ('pendiente','enviando','aceptado','aceptado_con_obs','rechazado','anulado','error')),
    sunat_codigo    TEXT,
    sunat_mensaje   TEXT,
    -- Artefactos
    xml_enviado     BYTEA,
    cdr_zip         BYTEA,
    hash_cpe        TEXT,
    pdf_a4          BYTEA,
    pdf_ticket      BYTEA,
    -- Datos completos del comprobante (líneas, descuentos, etc.) en JSON
    payload         JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, tipo_documento, serie, correlativo)
);
CREATE INDEX idx_comprobantes_tenant_fecha ON comprobantes(tenant_id, fecha_emision DESC);
CREATE INDEX idx_comprobantes_estado ON comprobantes(estado) WHERE estado IN ('pendiente','enviando','error');

-- ------------------------------------------------------------------------
-- Bitácora de envíos a SUNAT (un comprobante puede reintentarse N veces)
-- ------------------------------------------------------------------------
CREATE TABLE envios_sunat (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    comprobante_id  UUID NOT NULL REFERENCES comprobantes(id) ON DELETE CASCADE,
    intento         INT NOT NULL,
    iniciado_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    terminado_at    TIMESTAMPTZ,
    exito           BOOLEAN,
    codigo          TEXT,
    mensaje         TEXT,
    request_xml     BYTEA,
    response_xml    BYTEA
);
CREATE INDEX idx_envios_comprobante ON envios_sunat(comprobante_id);
