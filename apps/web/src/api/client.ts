const BASE = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

type Tokens = { access: string; refresh: string };

const TOKENS_KEY = "facturador.tokens";

export function getTokens(): Tokens | null {
  const raw = localStorage.getItem(TOKENS_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as Tokens;
  } catch {
    return null;
  }
}

export function setTokens(t: Tokens | null) {
  if (!t) localStorage.removeItem(TOKENS_KEY);
  else localStorage.setItem(TOKENS_KEY, JSON.stringify(t));
}

export class ApiError extends Error {
  status: number;
  body: any;
  constructor(status: number, body: any) {
    super(body?.detalle || body?.error || `HTTP ${status}`);
    this.status = status;
    this.body = body;
  }
}

async function refreshAccess(): Promise<string | null> {
  const t = getTokens();
  if (!t?.refresh) return null;
  const r = await fetch(`${BASE}/api/v1/auth/refresh`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: t.refresh }),
  });
  if (!r.ok) {
    setTokens(null);
    return null;
  }
  const data = await r.json();
  setTokens({ access: data.access_token, refresh: data.refresh_token });
  return data.access_token as string;
}

export async function api<T = any>(
  path: string,
  opts: { method?: string; body?: any; raw?: boolean } = {}
): Promise<T> {
  const tokens = getTokens();
  const headers: Record<string, string> = {};
  if (opts.body !== undefined) headers["Content-Type"] = "application/json";
  if (tokens?.access) headers["Authorization"] = `Bearer ${tokens.access}`;

  const doFetch = (h: Record<string, string>) =>
    fetch(`${BASE}${path}`, {
      method: opts.method || (opts.body !== undefined ? "POST" : "GET"),
      headers: h,
      body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
    });

  let resp = await doFetch(headers);
  if (resp.status === 401 && tokens?.refresh) {
    const newAccess = await refreshAccess();
    if (newAccess) {
      headers["Authorization"] = `Bearer ${newAccess}`;
      resp = await doFetch(headers);
    }
  }

  if (opts.raw) return resp as unknown as T;

  const text = await resp.text();
  let parsed: any;
  try {
    parsed = text ? JSON.parse(text) : null;
  } catch {
    parsed = text;
  }
  if (!resp.ok) throw new ApiError(resp.status, parsed);
  return parsed as T;
}

export async function apiBlob(path: string): Promise<Blob> {
  const tokens = getTokens();
  const headers: Record<string, string> = {};
  if (tokens?.access) headers["Authorization"] = `Bearer ${tokens.access}`;
  const r = await fetch(`${BASE}${path}`, { headers });
  if (!r.ok) throw new ApiError(r.status, await r.text());
  return r.blob();
}
