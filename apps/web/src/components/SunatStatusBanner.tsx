import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import { useAuth } from "../auth/AuthContext";

type EndpointStatus = {
  endpoint: string;
  up: boolean;
  last_error?: string;
  last_check_at: string;
  changed_at: string;
};

type StatusResp = {
  beta: EndpointStatus;
  prod: EndpointStatus;
  any_down: boolean;
  now: string;
};

export function SunatStatusBanner() {
  const { user, tenants } = useAuth();
  const [status, setStatus] = useState<StatusResp | null>(null);

  useEffect(() => {
    let cancelled = false;
    async function fetchStatus() {
      try {
        const r = await api<StatusResp>("/api/v1/sunat/status");
        if (!cancelled) setStatus(r);
      } catch {
        // silencioso: si la API no responde, el banner queda oculto y
        // el usuario verá el error del propio request que esté haciendo
      }
    }
    fetchStatus();
    const t = window.setInterval(fetchStatus, 30_000);
    return () => {
      cancelled = true;
      window.clearInterval(t);
    };
  }, []);

  if (!status) return null;

  // Si la empresa actual está en DEMO, SUNAT no nos importa: el otro
  // banner amarillo de DEMO ya explica que no se está enviando nada.
  const current = tenants.find((t) => t.id === user?.tenant_id);
  if (current?.demo_mode) return null;

  // Banner solo cuando el endpoint que estamos usando está caído.
  const relevant =
    current?.sunat_mode === "prod" ? status.prod : status.beta;
  if (relevant.up) return null;

  const fechaCambio = new Date(relevant.changed_at).toLocaleString("es-PE", {
    hour: "2-digit",
    minute: "2-digit",
    day: "2-digit",
    month: "2-digit",
  });

  return (
    <div className="bg-rose-50 border-b border-rose-200 text-rose-900 px-6 py-2 text-sm flex items-center justify-between gap-4">
      <div className="flex items-start gap-3">
        <span className="inline-block h-2 w-2 rounded-full bg-rose-500 mt-1.5 animate-pulse shrink-0" />
        <div>
          <div className="font-semibold">
            SUNAT no está respondiendo ({relevant.endpoint === "bill_prod" ? "producción" : "homologación"})
          </div>
          <div className="text-xs text-rose-800 mt-0.5">
            Detectado desde {fechaCambio}. Los comprobantes que emitas quedan
            en cola y se reintentan automáticamente cuando SUNAT vuelva.
            {relevant.last_error ? ` · ${relevant.last_error}` : null}
          </div>
        </div>
      </div>
      {user && (
        <Link
          to="/comprobantes?estado=error"
          className="shrink-0 text-rose-900 underline hover:no-underline text-xs whitespace-nowrap"
        >
          Ver afectados →
        </Link>
      )}
    </div>
  );
}
