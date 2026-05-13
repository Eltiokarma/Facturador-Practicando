import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { Toaster } from "sonner";
import { App } from "./App";
import { AuthProvider } from "./auth/AuthContext";
import { setupMobileAppearance } from "./services/mobile";
import "./index.css";

setupMobileAppearance();

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
