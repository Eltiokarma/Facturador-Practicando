-- Multi-tenant: relación many-to-many usuarios↔tenants.
--
-- Hasta ahora un usuario pertenecía a un solo tenant (users.tenant_id).
-- Para soportar el caso del contador con varios clientes o del dueño con
-- varias empresas, agregamos user_tenants(user_id, tenant_id, rol).
--
-- users.tenant_id se mantiene como "tenant por defecto al loguearse"
-- (último usado, o el primero si solo hay uno).

CREATE TABLE user_tenants (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    rol        TEXT NOT NULL CHECK (rol IN ('dueno','contador','cajero')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, tenant_id)
);
CREATE INDEX idx_user_tenants_user ON user_tenants(user_id);
CREATE INDEX idx_user_tenants_tenant ON user_tenants(tenant_id);

-- Migrar las membresías existentes del modelo viejo.
INSERT INTO user_tenants (user_id, tenant_id, rol)
SELECT id, tenant_id, rol FROM users
ON CONFLICT DO NOTHING;
