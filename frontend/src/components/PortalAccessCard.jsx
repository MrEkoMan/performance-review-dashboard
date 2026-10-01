import { useEffect, useState } from "react";
import { KeyRound, ShieldOff, RefreshCw } from "lucide-react";

import {
    getPortalAccess,
    createPortalAccess,
    resetPortalAccess,
    revokePortalAccess,
} from "../api/performanceApi.js";

// generatePassword produces a readable random password for the manager to
// hand over out-of-band. It is shown once at creation time and never
// retrievable afterwards.
function generatePassword() {
    const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789";
    const values = new Uint32Array(12);
    crypto.getRandomValues(values);
    return Array.from(values, (value) => alphabet[value % alphabet.length]).join("");
}

function PortalAccessCard({ engineerId }) {
    const [access, setAccess] = useState(null);
    const [loading, setLoading] = useState(true);
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");
    const [generated, setGenerated] = useState("");
    const [resetPassword, setResetPassword] = useState("");
    const [resetGenerated, setResetGenerated] = useState("");
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState("");
    const [notice, setNotice] = useState("");

    async function loadAccess() {
        try {
            setLoading(true);
            setError("");
            const status = await getPortalAccess(engineerId);
            setAccess(status || { hasAccess: false });
        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    }

    useEffect(() => {
        loadAccess();
        // loadAccess closes over engineerId.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [engineerId]);

    async function handleCreate(event) {
        event.preventDefault();
        try {
            setSaving(true);
            setError("");
            setNotice("");
            await createPortalAccess(engineerId, { email: email.trim(), password });
            setGenerated(password);
            setPassword("");
            setEmail("");
            await loadAccess();
            setNotice(
                "Portal access created. Share the password with the engineer " +
                "out-of-band — it cannot be retrieved again.",
            );
        } catch (err) {
            setError(err.message);
        } finally {
            setSaving(false);
        }
    }

    async function handleReset(event) {
        event.preventDefault();
        try {
            setSaving(true);
            setError("");
            setNotice("");
            await resetPortalAccess(engineerId, { password: resetPassword });
            setResetGenerated(resetPassword);
            setResetPassword("");
            await loadAccess();
            setNotice(
                "Password reset. Existing sessions were signed out; share the " +
                "new password with the engineer out-of-band.",
            );
        } catch (err) {
            setError(err.message);
        } finally {
            setSaving(false);
        }
    }

    async function handleRevoke() {
        if (!window.confirm("Revoke portal access? The engineer will be signed out and unable to log in.")) {
            return;
        }
        try {
            setSaving(true);
            setError("");
            setNotice("");
            await revokePortalAccess(engineerId);
            setResetGenerated("");
            await loadAccess();
            setNotice("Portal access revoked.");
        } catch (err) {
            setError(err.message);
        } finally {
            setSaving(false);
        }
    }

    if (loading) {
        return (
            <article className="profile-card portal-access-card">
                <h2>Portal Access</h2>
                <p>Loading...</p>
            </article>
        );
    }

    return (
        <article className="profile-card portal-access-card">
            <h2><KeyRound size={16} /> Portal Access</h2>

            {error && <div className="error">Error: {error}</div>}
            {notice && <div className="portal-access-notice">{notice}</div>}
            {generated && (
                <div className="portal-access-generated">
                    One-time password: <code>{generated}</code>
                </div>
            )}
            {resetGenerated && (
                <div className="portal-access-generated">
                    New password: <code>{resetGenerated}</code>
                </div>
            )}

            {access?.hasAccess ? (
                <>
                    <p>
                        Signed in as <strong>{access.email}</strong>. Password
                        resets sign out existing sessions immediately.
                    </p>
                    <form className="portal-access-form" onSubmit={handleReset}>
                        <label>
                            New password
                            <input
                                type="text"
                                value={resetPassword}
                                onChange={(event) => setResetPassword(event.target.value)}
                                placeholder="At least 8 characters"
                                autoComplete="off"
                                required
                            />
                        </label>
                        <button
                            type="button"
                            className="secondary-button"
                            onClick={() => setResetPassword(generatePassword())}
                            disabled={saving}
                        >
                            Generate
                        </button>
                        <button type="submit" disabled={saving || resetPassword.length < 8}>
                            <RefreshCw size={14} /> Reset password
                        </button>
                        <button
                            type="button"
                            className="secondary-button danger"
                            onClick={handleRevoke}
                            disabled={saving}
                        >
                            <ShieldOff size={14} /> Revoke access
                        </button>
                    </form>
                </>
            ) : (
                <>
                    <p>
                        No portal access yet. Grant this engineer a login so they
                        can view their own profile and add context for you.
                    </p>
                    <form className="portal-access-form" onSubmit={handleCreate}>
                        <label>
                            Email
                            <input
                                type="email"
                                value={email}
                                onChange={(event) => setEmail(event.target.value)}
                                placeholder="engineer@example.com"
                                required
                            />
                        </label>
                        <label>
                            Initial password
                            <input
                                type="text"
                                value={password}
                                onChange={(event) => setPassword(event.target.value)}
                                placeholder="At least 8 characters"
                                autoComplete="off"
                                required
                            />
                        </label>
                        <button
                            type="button"
                            className="secondary-button"
                            onClick={() => setPassword(generatePassword())}
                            disabled={saving}
                        >
                            Generate
                        </button>
                        <button type="submit" disabled={saving || !email.trim() || password.length < 8}>
                            Grant access
                        </button>
                    </form>
                </>
            )}
        </article>
    );
}

export default PortalAccessCard;
