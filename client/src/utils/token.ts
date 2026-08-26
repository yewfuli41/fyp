// The server signs a standard JWT carrying a numeric `exp` claim (see
// authService.GenerateToken). Reading it client-side lets an already-expired
// token be treated as logged-out on the very first render, instead of only
// once some request comes back 401 and fires the "unauthorized" event — which
// otherwise flashes a dashboard the user isn't actually authenticated for.
//
// This is a convenience check only: the payload is read without verifying the
// signature, so it can tell us "definitely expired" but never "definitely
// valid". The server still authoritatively rejects bad or expired tokens.
interface TokenPayload {
    exp?: number;
}

function decodePayload(token: string): TokenPayload | null {
    const payload = token.split(".")[1];
    if (!payload) return null;
    try {
        // JWTs use base64url, which atob doesn't accept as-is.
        const base64 = payload.replace(/-/g, "+").replace(/_/g, "/");
        return JSON.parse(atob(base64));
    } catch {
        return null;
    }
}

// Treats an unparseable token as expired — if we can't read it, the server
// won't accept it either.
export const isTokenExpired = (token: string | null): boolean => {
    if (!token) return true;
    const payload = decodePayload(token);
    if (!payload?.exp) return true;
    return payload.exp * 1000 <= Date.now();
};
