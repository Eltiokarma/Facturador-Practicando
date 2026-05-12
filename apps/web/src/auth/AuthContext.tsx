import {
  createContext,
  ReactNode,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import { api, getTokens, setTokens } from "../api/client";

export type User = {
  id: string;
  tenant_id: string;
  email: string;
  nombre: string;
  rol: "dueno" | "contador" | "cajero";
};

export type TenantSummary = {
  id: string;
  ruc: string;
  razon_social: string;
  sunat_mode: "beta" | "prod";
  has_cert: boolean;
};

type AuthState = {
  user: User | null;
  tenants: TenantSummary[];
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  switchTenant: (id: string) => Promise<void>;
  reloadTenants: () => Promise<void>;
};

const Ctx = createContext<AuthState | undefined>(undefined);

async function loadMe(): Promise<{ user: User; tenants: TenantSummary[] }> {
  const [user, tenantsResp] = await Promise.all([
    api<User>("/api/v1/me"),
    api<{ items: TenantSummary[] }>("/api/v1/me/tenants"),
  ]);
  return { user, tenants: tenantsResp.items || [] };
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [tenants, setTenants] = useState<TenantSummary[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const t = getTokens();
    if (!t?.access) {
      setLoading(false);
      return;
    }
    loadMe()
      .then(({ user, tenants }) => {
        setUser(user);
        setTenants(tenants);
      })
      .catch(() => {
        setTokens(null);
        setUser(null);
        setTenants([]);
      })
      .finally(() => setLoading(false));
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const data = await api<{
      access_token: string;
      refresh_token: string;
      user: User;
    }>("/api/v1/auth/login", { body: { email, password } });
    setTokens({ access: data.access_token, refresh: data.refresh_token });
    setUser(data.user);
    // Cargar tenants disponibles
    try {
      const r = await api<{ items: TenantSummary[] }>("/api/v1/me/tenants");
      setTenants(r.items || []);
    } catch {
      setTenants([]);
    }
  }, []);

  const logout = useCallback(() => {
    setTokens(null);
    setUser(null);
    setTenants([]);
  }, []);

  const switchTenant = useCallback(async (tenantID: string) => {
    const data = await api<{
      access_token: string;
      refresh_token: string;
      user: User;
    }>("/api/v1/auth/switch-tenant", { body: { tenant_id: tenantID } });
    setTokens({ access: data.access_token, refresh: data.refresh_token });
    setUser(data.user);
  }, []);

  const reloadTenants = useCallback(async () => {
    try {
      const r = await api<{ items: TenantSummary[] }>("/api/v1/me/tenants");
      setTenants(r.items || []);
    } catch {
      // best effort
    }
  }, []);

  return (
    <Ctx.Provider
      value={{ user, tenants, loading, login, logout, switchTenant, reloadTenants }}
    >
      {children}
    </Ctx.Provider>
  );
}

export function useAuth(): AuthState {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("useAuth fuera de AuthProvider");
  return ctx;
}
