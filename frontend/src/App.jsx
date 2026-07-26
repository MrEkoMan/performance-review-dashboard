import { Route, Routes } from "react-router-dom";
import { useEffect, useMemo, useState } from "react";
import { CssBaseline, ThemeProvider } from "@mui/material";

import DashboardPage from "./pages/DashboardPage.jsx";
import EngineeringProfilePage from "./pages/EngineerProfilePage.jsx";
import SettingsPage from "./pages/SettingsPage.jsx";

import { getSettings } from "./api/performanceApi.js";
import AppShell from "./components/AppShell.jsx";
import { buildTheme, ThemeModeContext } from "./theme.jsx";

function App() {
  const [mode, setMode] = useState("light");
  const theme = useMemo(() => buildTheme(mode), [mode]);

  useEffect(() => {
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
  }, []);

  return (
    <ThemeModeContext.Provider value={{ mode, setMode }}>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        <AppShell>
          <Routes>
            <Route path="/" element={<DashboardPage />} />

            <Route
              path="/engineers/:engineerId"
              element={<EngineeringProfilePage />}
            />

            <Route path="/settings" element={<SettingsPage />} />

            <Route
              path="*"
              element={<h1>Page not found</h1>}
            />
          </Routes>
        </AppShell>
      </ThemeProvider>
    </ThemeModeContext.Provider>
  )
}

export default App;
