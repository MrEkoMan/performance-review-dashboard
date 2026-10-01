import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Box, Button, Card, CardContent, TextField, Typography, Alert, Divider } from "@mui/material";

import { login, registerAccount, getRegistrationStatus } from "../api/performanceApi.js";
import { useAuth } from "../context/useAuth.js";

function LoginPage() {
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");
    const navigate = useNavigate();
    const { setUser } = useAuth();

    // Registration claiming: when no accounts exist (or no manager does), the
    // login page offers to create the manager account so the operator is
    // never locked out.
    const [canClaimManager, setCanClaimManager] = useState(false);
    const [regEmail, setRegEmail] = useState("");
    const [regPassword, setRegPassword] = useState("");
    const [regConfirm, setRegConfirm] = useState("");
    const [registering, setRegistering] = useState(false);
    const [registerError, setRegisterError] = useState("");

    useEffect(() => {
        let cancelled = false;
        async function loadStatus() {
            try {
                const status = await getRegistrationStatus();
                if (!cancelled) {
                    setCanClaimManager(Boolean(status?.canClaimManager));
                }
            } catch {
                // Backend unreachable or older version — keep the login-only form.
                if (!cancelled) {
                    setCanClaimManager(false);
                }
            }
        }
        loadStatus();
        return () => {
            cancelled = true;
        };
    }, []);

    function finishSignIn(user) {
        setUser(user);
        if (user?.role === "engineer" && user?.engineerId) {
            navigate(`/engineers/${user.engineerId}`, { replace: true });
        } else {
            navigate("/", { replace: true });
        }
    }

    async function handleSubmit(event) {
        event.preventDefault();
        if (!email.trim() || !password) {
            setError("Email and password are required.");
            return;
        }
        try {
            setSubmitting(true);
            setError("");
            const user = await login(email.trim(), password);
            finishSignIn(user);
        } catch (err) {
            setError(err.message || "Login failed.");
        } finally {
            setSubmitting(false);
        }
    }

    async function handleRegister(event) {
        event.preventDefault();
        if (!regEmail.trim() || regPassword.length < 8) {
            setRegisterError("Email and a password of at least 8 characters are required.");
            return;
        }
        if (regPassword !== regConfirm) {
            setRegisterError("Passwords do not match.");
            return;
        }
        try {
            setRegistering(true);
            setRegisterError("");
            const user = await registerAccount({
                email: regEmail.trim(),
                password: regPassword,
            });
            finishSignIn(user);
        } catch (err) {
            setRegisterError(err.message || "Registration failed.");
        } finally {
            setRegistering(false);
        }
    }

    return (
        <main className="login-page">
            <Card className="login-card" elevation={3}>
                <CardContent>
                    <Typography variant="h5" component="h1" gutterBottom>
                        Engineering Manager OS
                    </Typography>

                    {canClaimManager ? (
                        <>
                            <Typography variant="body2" color="text.secondary" gutterBottom>
                                No manager account exists yet. Create one to start
                                using the dashboard — the first account becomes the
                                administrator.
                            </Typography>
                            {registerError && (
                                <Alert severity="error" sx={{ mb: 2 }}>{registerError}</Alert>
                            )}
                            <Box component="form" onSubmit={handleRegister} noValidate>
                                <TextField
                                    label="Admin email"
                                    type="email"
                                    value={regEmail}
                                    onChange={(event) => setRegEmail(event.target.value)}
                                    fullWidth
                                    margin="normal"
                                    autoComplete="username"
                                    autoFocus
                                    required
                                />
                                <TextField
                                    label="Admin password"
                                    type="password"
                                    value={regPassword}
                                    onChange={(event) => setRegPassword(event.target.value)}
                                    fullWidth
                                    margin="normal"
                                    autoComplete="new-password"
                                    required
                                />
                                <TextField
                                    label="Confirm password"
                                    type="password"
                                    value={regConfirm}
                                    onChange={(event) => setRegConfirm(event.target.value)}
                                    fullWidth
                                    margin="normal"
                                    autoComplete="new-password"
                                    required
                                />
                                <Button
                                    type="submit"
                                    variant="contained"
                                    fullWidth
                                    disabled={registering}
                                    sx={{ mt: 2 }}
                                >
                                    {registering ? "Creating account..." : "Create manager account"}
                                </Button>
                            </Box>
                            <Divider sx={{ my: 2 }}>
                                <Typography variant="caption" color="text.secondary">
                                    already have an account?
                                </Typography>
                            </Divider>
                        </>
                    ) : (
                        <>
                            <Typography variant="body2" color="text.secondary" gutterBottom>
                                Sign in to your account.
                            </Typography>
                            {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
                            <Box component="form" onSubmit={handleSubmit} noValidate>
                                <TextField
                                    label="Email"
                                    type="email"
                                    value={email}
                                    onChange={(event) => setEmail(event.target.value)}
                                    fullWidth
                                    margin="normal"
                                    autoComplete="username"
                                    autoFocus
                                    required
                                />
                                <TextField
                                    label="Password"
                                    type="password"
                                    value={password}
                                    onChange={(event) => setPassword(event.target.value)}
                                    fullWidth
                                    margin="normal"
                                    autoComplete="current-password"
                                    required
                                />
                                <Button
                                    type="submit"
                                    variant="contained"
                                    fullWidth
                                    disabled={submitting}
                                    sx={{ mt: 2 }}
                                >
                                    {submitting ? "Signing in..." : "Sign in"}
                                </Button>
                            </Box>
                        </>
                    )}
                </CardContent>
            </Card>
        </main>
    );
}

export default LoginPage;
