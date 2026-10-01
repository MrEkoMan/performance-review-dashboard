import { useState } from "react";
import { Link as RouterLink, useLocation, useNavigate } from "react-router-dom";
import {
  AppBar,
  Box,
  Button,
  Drawer,
  IconButton,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Toolbar,
  Typography,
} from "@mui/material";
import DashboardOutlinedIcon from "@mui/icons-material/DashboardOutlined";
import LogoutOutlinedIcon from "@mui/icons-material/LogoutOutlined";
import MenuIcon from "@mui/icons-material/Menu";
import PersonOutlinedIcon from "@mui/icons-material/PersonOutlined";
import SettingsOutlinedIcon from "@mui/icons-material/SettingsOutlined";

import { useAuth } from "../context/useAuth.js";
import { logout } from "../api/performanceApi.js";

const drawerWidth = 240;

function Navigation({ onNavigate }) {
  const location = useLocation();
  const { user } = useAuth();
  const isEngineer = user?.role === "engineer";

  if (isEngineer) {
    return (
      <List sx={{ px: 1 }}>
        <ListItemButton
          component={RouterLink}
          to={`/engineers/${user.engineerId}`}
          selected={location.pathname.startsWith("/engineers/")}
          onClick={onNavigate}
        >
          <ListItemIcon><PersonOutlinedIcon /></ListItemIcon>
          <ListItemText primary="My Profile" />
        </ListItemButton>
      </List>
    );
  }

  return (
    <List sx={{ px: 1 }}>
      <ListItemButton
        component={RouterLink}
        to="/"
        selected={location.pathname === "/" || location.pathname.startsWith("/engineers/")}
        onClick={onNavigate}
      >
        <ListItemIcon><DashboardOutlinedIcon /></ListItemIcon>
        <ListItemText primary="Dashboard" />
      </ListItemButton>
      <ListItemButton
        component={RouterLink}
        to="/settings"
        selected={location.pathname === "/settings"}
        onClick={onNavigate}
      >
        <ListItemIcon><SettingsOutlinedIcon /></ListItemIcon>
        <ListItemText primary="Settings" />
      </ListItemButton>
    </List>
  );
}

function SessionBar() {
  const { user, setUser } = useAuth();
  const navigate = useNavigate();

  if (!user) {
    return null;
  }

  async function handleLogout() {
    try {
      await logout();
    } catch {
      // The session may already be gone; clear local state regardless.
    }
    setUser(null);
    navigate("/login", { replace: true });
  }

  return (
    <Box sx={{ ml: "auto", display: "flex", alignItems: "center", gap: 1 }}>
      <Typography variant="body2" color="text.secondary">
        {user.name || user.email}
      </Typography>
      <Button
        size="small"
        startIcon={<LogoutOutlinedIcon />}
        onClick={handleLogout}
      >
        Log out
      </Button>
    </Box>
  );
}

function AppShell({ children }) {
  const [mobileOpen, setMobileOpen] = useState(false);

  return (
    <Box sx={{ display: "flex", minHeight: "100vh" }}>
      <AppBar
        position="fixed"
        color="default"
        elevation={0}
        sx={{ borderBottom: 1, borderColor: "divider", zIndex: (theme) => theme.zIndex.drawer + 1 }}
      >
        <Toolbar>
          <IconButton
            edge="start"
            onClick={() => setMobileOpen(true)}
            aria-label="Open navigation"
            sx={{ mr: 1, display: { md: "none" } }}
          >
            <MenuIcon />
          </IconButton>
          <Typography variant="h6" component="div">
            Engineering Manager OS
          </Typography>
          <SessionBar />
        </Toolbar>
      </AppBar>

      <Drawer
        variant="temporary"
        open={mobileOpen}
        onClose={() => setMobileOpen(false)}
        ModalProps={{ keepMounted: true }}
        sx={{ display: { xs: "block", md: "none" }, "& .MuiDrawer-paper": { width: drawerWidth } }}
      >
        <Toolbar />
        <Navigation onNavigate={() => setMobileOpen(false)} />
      </Drawer>

      <Drawer
        variant="permanent"
        sx={{
          display: { xs: "none", md: "block" },
          width: drawerWidth,
          flexShrink: 0,
          "& .MuiDrawer-paper": { width: drawerWidth, boxSizing: "border-box" },
        }}
        open
      >
        <Toolbar />
        <Navigation />
      </Drawer>

      <Box component="div" role="main" sx={{ flexGrow: 1, minWidth: 0, pt: 8 }}>
        {children}
      </Box>
    </Box>
  );
}

export default AppShell;
