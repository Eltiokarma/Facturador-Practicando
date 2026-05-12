import {
  createContext,
  ReactNode,
  useContext,
  useEffect,
  useState,
  useCallback,
} from "react";
import { api, getTokens, setTokens } from "../api/client";

export type User = {
  id: string;
  tenant_id: string;
  email: string;
  nombre: string;
  rol: "dueno" | "contador" | "cajero";
};

type AuthState = {
  user: User | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
};

const Ctx = createContext<AuthState | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const t = getTokens();
    if (!t?.access) {
      setLoading(false);
      return;
    }
    api<User>("/api/v1/me")
      .then(setUser)
      .catch(() => {
        setTokens(null);
        setUser(null);
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
  }, []);

  const logout = useCallback(() => {
    setTokens(null);
    setUser(null);
  }, []);

  return (
    <Ctx.Provider value={{ user, loading, login, logout }}>{children}</Ctx.Provider>
  );
}

export function useAuth(): AuthState {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("useAuth fuera de AuthProvider");
  return ctx;
}
