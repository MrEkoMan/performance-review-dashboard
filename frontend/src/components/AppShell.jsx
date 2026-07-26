import { useState } from "react";
import { Link as RouterLink, useLocation } from "react-router-dom";
import {
  AppBar,
  Box,
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
import MenuIcon from "@mui/icons-material/Menu";
import SettingsOutlinedIcon from "@mui/icons-material/SettingsOutlined";

const drawerWidth = 240;

function Navigation({ onNavigate }) {
  const location = useLocation();

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
