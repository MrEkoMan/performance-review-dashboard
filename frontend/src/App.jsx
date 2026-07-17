import { Route, Routes } from "react-router";

import DashboardPage from "./pages/DashboardPage.jsx";
import EngineeringProfilePage from "./pages/EngineerProfilePage.jsx";

function App() {
  return (
    <Routes>
      <Route path="/" element={<DashboardPage />} />

      <Route
        path="/engineers/:engineerId"
        element={<EngineeringProfilePage />} 
      />

      <Route
        path="*"
        element={<h1>Page not found</h1>}
      />
    </Routes>
  )
}

export default App;