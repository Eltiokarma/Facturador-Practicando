import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { Toaster } from "sonner";
import { App } from "./App";
import { AuthProvider } from "./auth/AuthContext";
import { setupMobileAppearance } from "./services/mobile";
import { requestPermission as requestNotifPermission } from "./services/notifications";
import "./index.css";

setupMobileAppearance();
// No bloqueamos el render por esto. Si el usuario rechaza, no le llegan
// notificaciones; el toast in-app sigue funcionando igual.
requestNotifPermission().catch(() => { /* noop */ });

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <BrowserRouter>
      <AuthProvider>
        <App />
        <Toaster
          position="top-right"
          richColors
          closeButton
          toastOptions={{ className: "text-sm" }}
        />
      </AuthProvider>
    </BrowserRouter>
  </StrictMode>
);
