import { Route, Routes } from "react-router-dom";
import { useEffect } from "react";

import DashboardPage from "./pages/DashboardPage.jsx";
import EngineeringProfilePage from "./pages/EngineerProfilePage.jsx";
import SettingsPage from "./pages/SettingsPage.jsx";

import { getSettings } from "./api/performanceApi.js";

function App() {
  useEffect(() => {
    async function applySavedTheme() {
      try {
        const settings = await getSettings();
        const theme = settings?.theme || "light";

        document.documentElement.dataset.theme = theme;
      } catch (err) {
        console.error("Failed to load theme", err);
      }
    }
    
    applySavedTheme();
  }, []);

  return (
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
  )
}

export default App;