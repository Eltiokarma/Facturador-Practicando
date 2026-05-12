import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { TenantSwitcher } from "../components/TenantSwitcher";

const items = [
  { to: "/dashboard", label: "Resumen", icon: "📊" },
  { to: "/emitir", label: "Emitir comprobante", icon: "🧾" },
  { to: "/comprobantes", label: "Comprobantes", icon: "🗂️" },
  { to: "/resumenes", label: "Resumen diario", icon: "📅" },
  { to: "/clientes", label: "Clientes", icon: "👥" },
  { to: "/productos", label: "Productos", icon: "📦" },
  { to: "/configuracion", label: "Configuración", icon: "⚙️" },
];

export function Layout() {
  const { user, logout } = useAuth();
  const nav = useNavigate();

  return (
    <div className="min-h-screen flex bg-slate-50">
      <aside className="w-64 shrink-0 border-r border-slate-200 bg-white flex flex-col">
        <div className="px-4 py-4 border-b border-slate-100 space-y-3">
          <div>
            <div className="text-lg font-semibold text-slate-900">Facturador</div>
            <div className="text-xs text-slate-500 mt-0.5">self-hosted Perú</div>
          </div>
          <TenantSwitcher />
        </div>
        <nav className="flex-1 px-3 py-4 space-y-1">
          {items.map((it) => (
            <NavLink
              key={it.to}
              to={it.to}
              className={({ isActive }) =>
                `flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition ${
                  isActive
                    ? "bg-brand-50 text-brand-700"
                    : "text-slate-700 hover:bg-slate-50"
                }`
              }
            >
              <span aria-hidden>{it.icon}</span>
              {it.label}
            </NavLink>
          ))}
        </nav>
        <div className="px-3 py-4 border-t border-slate-100">
          <div className="px-3 py-2 text-xs">
            <div className="text-slate-900 font-medium truncate">
              {user?.nombre}
            </div>
            <div className="text-slate-500 truncate">{user?.email}</div>
            <div className="text-slate-400 mt-1 uppercase tracking-wide text-[10px]">
              {user?.rol}
            </div>
          </div>
          <button
            className="w-full mt-1 btn-ghost text-sm"
            onClick={() => {
              logout();
              nav("/login", { replace: true });
            }}
          >
            Cerrar sesión
          </button>
        </div>
      </aside>

      <main className="flex-1 min-w-0 overflow-x-hidden">
        <div className="max-w-5xl mx-auto px-8 py-10">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
