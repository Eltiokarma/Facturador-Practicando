import { useEffect, useState } from "react";

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

type Health = {
  status: string;
  sunat_mode: string;
  time: string;
};

export function App() {
  const [health, setHealth] = useState<Health | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch(`${API_BASE}/health`)
      .then((r) => r.json())
      .then(setHealth)
      .catch((e) => setError(String(e)));
  }, []);

  return (
    <main className="min-h-screen flex items-center justify-center p-6">
      <div className="max-w-xl w-full space-y-6">
        <header>
          <h1 className="text-3xl font-bold">Facturador</h1>
          <p className="text-slate-400">
            Facturador electrónico self-hosted Perú — scaffolding inicial.
          </p>
        </header>

        <section className="rounded-lg border border-slate-800 bg-slate-900 p-4">
          <h2 className="text-lg font-semibold mb-2">Estado de la API</h2>
          {error && <p className="text-red-400">No se pudo conectar: {error}</p>}
          {!error && !health && <p className="text-slate-400">Cargando…</p>}
          {health && (
            <ul className="text-sm space-y-1">
              <li>
                <span className="text-slate-400">status:</span> {health.status}
              </li>
              <li>
                <span className="text-slate-400">sunat_mode:</span>{" "}
                <span
                  className={
                    health.sunat_mode === "prod"
                      ? "text-amber-400"
                      : "text-emerald-400"
                  }
                >
                  {health.sunat_mode}
                </span>
              </li>
              <li>
                <span className="text-slate-400">time:</span> {health.time}
              </li>
            </ul>
          )}
        </section>

        <footer className="text-xs text-slate-500">
          Próximo paso: emitir 1 factura beta contra SUNAT homologación.
        </footer>
      </div>
    </main>
  );
}
