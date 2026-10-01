import { useContext } from "react";
import AuthContext from "./AuthContext.jsx";

// useAuth exposes { user, setUser, ready } from the AuthProvider. Kept in its
// own module so AuthContext.jsx only exports components (fast-refresh rule).
export function useAuth() {
    const context = useContext(AuthContext);
    if (!context) {
        throw new Error("useAuth must be used inside AuthProvider");
    }
    return context;
}
