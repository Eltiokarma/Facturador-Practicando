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
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
