-- Catálogos persistentes por tenant: clientes y productos.
-- Pensados para que el cajero NO escriba el RUC o la descripción cada vez.

-- ------------------------------------------------------------------------
-- Clientes
-- ------------------------------------------------------------------------
CREATE TABLE clientes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    tipo_doc     CHAR(1) NOT NULL,  -- catálogo SUNAT 06: 1 DNI, 6 RUC, etc.
    num_doc      TEXT NOT NULL,
    razon_social TEXT NOT NULL,
    direccion    TEXT,
    email        TEXT,
    telefono     TEXT,
    activo       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, tipo_doc, num_doc)
);
CREATE INDEX idx_clientes_tenant ON clientes(tenant_id) WHERE activo;
CREATE INDEX idx_clientes_busqueda ON clientes(tenant_id, num_doc text_pattern_ops);

-- ------------------------------------------------------------------------
-- Productos / servicios
-- ------------------------------------------------------------------------
CREATE TABLE productos (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    codigo          TEXT,             -- código interno del negocio (SKU); opcional
    descripcion     TEXT NOT NULL,
    unidad          CHAR(3) NOT NULL DEFAULT 'NIU',  -- catálogo SUNAT 03 (NIU, ZZ, KGM, ...)
    valor_unitario  NUMERIC(12,4) NOT NULL DEFAULT 0,  -- sin IGV
    afectacion_igv  CHAR(2) NOT NULL DEFAULT '10',     -- catálogo SUNAT 07
    activo          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_productos_tenant ON productos(tenant_id) WHERE activo;
CREATE INDEX idx_productos_busqueda ON productos(tenant_id, lower(descripcion) text_pattern_ops);
