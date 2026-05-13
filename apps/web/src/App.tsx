import { Navigate, Route, Routes } from "react-router-dom";
import { RequireAuth } from "./auth/RequireAuth";
import { Layout } from "./pages/Layout";
import { Login } from "./pages/Login";
import { Dashboard } from "./pages/Dashboard";
import { NuevaFactura } from "./pages/NuevaFactura";
import { Comprobantes } from "./pages/Comprobantes";
import { ComprobanteDetalle } from "./pages/ComprobanteDetalle";
import { Clientes } from "./pages/Clientes";
import { Productos } from "./pages/Productos";
import { Resumenes } from "./pages/Resumenes";
import { ResumenDetalle } from "./pages/ResumenDetalle";
import { Configuracion } from "./pages/Configuracion";
import { NuevaEmpresa } from "./pages/NuevaEmpresa";
import { NuevaGuia } from "./pages/NuevaGuia";
import { NuevaNota } from "./pages/NuevaNota";

export function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        path="/"
        element={
          <RequireAuth>
            <Layout />
          </RequireAuth>
        }
      >
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="dashboard" element={<Dashboard />} />
        <Route path="emitir" element={<NuevaFactura />} />
        <Route path="comprobantes" element={<Comprobantes />} />
        <Route path="comprobantes/:id" element={<ComprobanteDetalle />} />
        <Route path="clientes" element={<Clientes />} />
        <Route path="productos" element={<Productos />} />
        <Route path="resumenes" element={<Resumenes />} />
        <Route path="resumenes/:id" element={<ResumenDetalle />} />
        <Route path="configuracion" element={<Configuracion />} />
        <Route path="empresas/nueva" element={<NuevaEmpresa />} />
        <Route path="notas/:tipo" element={<NuevaNota />} />
        <Route path="guias/nueva" element={<NuevaGuia />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
