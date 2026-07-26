import { createContext, useContext } from "react";
import { createTheme } from "@mui/material/styles";

export const ThemeModeContext = createContext({
  mode: "light",
  setMode: () => {},
});

export function useThemeMode() {
  return useContext(ThemeModeContext);
}

export function buildTheme(mode) {
  return createTheme({
    palette: {
      mode,
      primary: {
        main: mode === "dark" ? "#a8c7fa" : "#0b57d0",
      },
      secondary: {
        main: mode === "dark" ? "#c2e7ff" : "#00639b",
      },
      background: {
        default: mode === "dark" ? "#111318" : "#f8f9ff",
        paper: mode === "dark" ? "#1b1b1f" : "#ffffff",
      },
    },
    shape: {
      borderRadius: 12,
    },
    typography: {
      fontFamily: '"Roboto", "Segoe UI", sans-serif',
      h1: { fontSize: "2rem", fontWeight: 500 },
      h2: { fontSize: "1.5rem", fontWeight: 500 },
      button: { textTransform: "none", fontWeight: 600 },
    },
    components: {
      MuiCard: {
        defaultProps: { variant: "outlined" },
      },
      MuiButton: {
        defaultProps: { disableElevation: true },
      },
    },
  });
}
