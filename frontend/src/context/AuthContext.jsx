import { createContext, useEffect, useMemo, useState } from "react";
import { getMe } from "../api/performanceApi.js";

const AuthContext = createContext(null);

// AuthContext exposes the logged-in account (or null) once the session has
// been resolved. `ready` is false until the initial /auth/me lookup finishes
// so route guards do not bounce a logged-in user to /login on first paint.
export function AuthProvider({ children }) {
    const [user, setUser] = useState(null);
    const [ready, setReady] = useState(false);

    useEffect(() => {
        let cancelled = false;
        async function resolve() {
            try {
                const me = await getMe();
                if (!cancelled) {
                    setUser(me || null);
                }
            } catch {
                // Not logged in (or session expired) — stay anonymous.
                if (!cancelled) {
                    setUser(null);
                }
            } finally {
                if (!cancelled) {
                    setReady(true);
                }
            }
        }
        resolve();
        return () => {
            cancelled = true;
        };
    }, []);

    const value = useMemo(
        () => ({ user, setUser, ready }),
        [user, ready],
    );

    return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export default AuthContext;
