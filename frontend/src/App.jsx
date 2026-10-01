import { Route, Routes, Navigate } from "react-router-dom";
import { useEffect, useMemo, useState } from "react";
import { CssBaseline, ThemeProvider } from "@mui/material";

import DashboardPage from "./pages/DashboardPage.jsx";
import EngineeringProfilePage from "./pages/EngineerProfilePage.jsx";
import SettingsPage from "./pages/SettingsPage.jsx";
import LoginPage from "./pages/LoginPage.jsx";

import { getSettings } from "./api/performanceApi.js";
import AppShell from "./components/AppShell.jsx";
import { AuthProvider } from "./context/AuthContext.jsx";
import { useAuth } from "./context/useAuth.js";
import { buildTheme, ThemeModeContext } from "./theme.jsx";

function RequireAuth({ children, user, ready }) {
  if (!ready) {
    return null;
  }
  if (!user) {
    return <Navigate to="/login" replace />;
  }
  return children;
}

function App() {
  const [mode, setMode] = useState("light");
  const theme = useMemo(() => buildTheme(mode), [mode]);
  const { user, ready } = useAuth();

  useEffect(() => {
    // Theme settings are manager-gated on the API; an engineer session (or a
    // signed-out visitor) simply keeps the light default.
    if (!ready || !user || user.role !== "manager") {
      return;
    }
    async function applySavedTheme() {
      try {
        const settings = await getSettings();
        const theme = settings?.theme || "light";

        document.documentElement.dataset.theme = theme;
        setMode(theme);
      } catch (err) {
        console.error("Failed to load theme", err);
      }
    }

    applySavedTheme();
  }, [ready, user]);

  if (!ready) {
    return null;
  }

  return (
    <ThemeModeContext.Provider value={{ mode, setMode }}>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        <Routes>
          <Route path="/login" element={<LoginPage />} />

          <Route
            path="*"
            element={
              <RequireAuth user={user} ready={ready}>
                <AppShell>
                  <AuthRoutes />
                </AppShell>
              </RequireAuth>
            }
          />
        </Routes>
      </ThemeProvider>
    </ThemeModeContext.Provider>
  )
}

// AuthRoutes holds the authenticated route table. Engineers are redirected
// from the dashboard to their own profile so they never see the manager view.
function AuthRoutes() {
  const { user } = useAuth();

  return (
    <Routes>
      <Route
        path="/"
        element={
          user?.role === "engineer" && user?.engineerId
            ? <Navigate to={`/engineers/${user.engineerId}`} replace />
            : <DashboardPage />
        }
      />

      <Route
        path="/engineers/:engineerId"
        element={<EngineeringProfilePage />}
      />

      <Route
        path="/settings"
        element={user?.role === "manager" ? <SettingsPage /> : <Navigate to="/" replace />}
      />

      <Route
        path="*"
        element={<h1>Page not found</h1>}
      />
    </Routes>
  );
}

export default App;
