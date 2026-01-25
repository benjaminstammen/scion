# OAuth Implementation Progress Report

**Date:** 2026-01-25
**Task:** Milestone 4 - Authentication Flow (OAuth Setup Documentation)
**Status:** Documentation created, needs to be re-committed

---

## Summary

Created comprehensive OAuth setup documentation at `.design/oauth-setup.md` with step-by-step instructions for configuring Google and GitHub OAuth providers.

---

## File Created

### `.design/oauth-setup.md`

Complete OAuth configuration guide covering:

---

## Section 1: Environment Variables Reference

```bash
# Required for all OAuth flows
SESSION_SECRET=<random-32-character-string>
BASE_URL=http://localhost:8080  # Or production URL

# Google OAuth (optional if using GitHub)
GOOGLE_CLIENT_ID=<your-google-client-id>
GOOGLE_CLIENT_SECRET=<your-google-client-secret>

# GitHub OAuth (optional if using Google)
GITHUB_CLIENT_ID=<your-github-client-id>
GITHUB_CLIENT_SECRET=<your-github-client-secret>

# Authorization (optional)
AUTHORIZED_DOMAINS=example.com,company.org  # Comma-separated list
```

---

## Section 2: Google OAuth Setup (Detailed Steps)

### Step 1: Access Google Cloud Console
- Navigate to https://console.cloud.google.com/
- Sign in with appropriate Google account

### Step 2: Create or Select Project
- Click project dropdown at top
- Create new project named `scion-web` or select existing
- Wait for project creation notification

### Step 3: Configure OAuth Consent Screen
1. Navigate to **APIs & Services** → **OAuth consent screen**
2. Select user type:
   - **Internal**: Only Google Workspace org users (recommended for internal tools)
   - **External**: Any Google user (requires verification for production)
3. Fill in consent screen form:
   - App name: `Scion`
   - User support email: Your email
   - Authorized domains: Add production domain
   - Developer contact: Your email
4. Add scopes:
   - `email` - View email address
   - `profile` - View basic profile info
   - `openid` - Associate with personal info
5. Add test users (for External apps only)

### Step 4: Create OAuth Credentials
1. Navigate to **APIs & Services** → **Credentials**
2. Click **+ Create Credentials** → **OAuth client ID**
3. Configure:
   - Application type: **Web application**
   - Name: `Scion Web Frontend`
4. Add **Authorized JavaScript origins**:
   ```
   http://localhost:8080
   https://your-production-domain.com
   ```
5. Add **Authorized redirect URIs**:
   ```
   http://localhost:8080/auth/callback/google
   https://your-production-domain.com/auth/callback/google
   ```
6. Click Create and copy:
   - Client ID → `GOOGLE_CLIENT_ID`
   - Client Secret → `GOOGLE_CLIENT_SECRET`

---

## Section 3: GitHub OAuth Setup (Detailed Steps)

### Step 1: Access GitHub Developer Settings
- Go to GitHub → Profile → Settings → Developer settings → OAuth Apps

### Step 2: Create New OAuth App
1. Click "New OAuth App"
2. Fill in form:
   - Application name: `Scion`
   - Homepage URL: `http://localhost:8080`
   - Authorization callback URL: `http://localhost:8080/auth/callback/github`
3. Click "Register application"

### Step 3: Get Credentials
- Copy **Client ID** → `GITHUB_CLIENT_ID`
- Click "Generate a new client secret"
- Copy secret → `GITHUB_CLIENT_SECRET`

### Note on Multiple Environments
- GitHub only allows one callback URL per OAuth app
- Create separate OAuth apps for dev, staging, prod

---

## Section 4: Session Secret Generation

Three methods provided:

**OpenSSL (recommended):**
```bash
openssl rand -base64 32
```

**Node.js:**
```bash
node -e "console.log(require('crypto').randomBytes(32).toString('base64'))"
```

**Python:**
```bash
python3 -c "import secrets; print(secrets.token_urlsafe(32))"
```

Requirements:
- At least 32 characters
- Cryptographically random
- Never committed to version control

---

## Section 5: Authorization Configuration

Domain-based authorization via `AUTHORIZED_DOMAINS`:

```bash
# Allow specific domains
AUTHORIZED_DOMAINS=example.com,mycompany.org

# Allow all domains (dev only, not for production)
AUTHORIZED_DOMAINS=*
```

**How it works:**
1. User authenticates via OAuth
2. Frontend extracts email domain
3. Domain checked against AUTHORIZED_DOMAINS
4. Access granted if match found, otherwise "Unauthorized" error

---

## Section 6: Complete .env Template

```bash
# Server configuration
PORT=8080
NODE_ENV=development

# Hub API
HUB_API_URL=http://localhost:9810

# Session
SESSION_SECRET=your-32-character-or-longer-secret-here

# OAuth - Base URL for callbacks
BASE_URL=http://localhost:8080

# Google OAuth
GOOGLE_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-google-client-secret

# GitHub OAuth
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret

# Authorization
AUTHORIZED_DOMAINS=example.com,mycompany.org
```

---

## Section 7: Production Configuration (Cloud Run)

**Secret Manager creation:**
```bash
echo -n "your-session-secret" | gcloud secrets create session-secret --data-file=-
echo -n "your-google-client-id" | gcloud secrets create google-client-id --data-file=-
echo -n "your-google-client-secret" | gcloud secrets create google-client-secret --data-file=-
```

**Cloud Run YAML reference:**
```yaml
env:
  - name: SESSION_SECRET
    valueFrom:
      secretKeyRef:
        name: scion-secrets
        key: session-secret
  - name: GOOGLE_CLIENT_ID
    valueFrom:
      secretKeyRef:
        name: scion-secrets
        key: google-client-id
  - name: GOOGLE_CLIENT_SECRET
    valueFrom:
      secretKeyRef:
        name: scion-secrets
        key: google-client-secret
```

---

## Section 8: Troubleshooting Tables

### Google OAuth Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `redirect_uri_mismatch` | Callback URL doesn't match | Verify redirect URI exactly matches `BASE_URL/auth/callback/google` |
| `invalid_client` | Wrong client ID/secret | Double-check credentials, no extra whitespace |
| `access_denied` | User denied consent | User clicked "Cancel" |
| `org_internal` | External user on internal app | Change to "External" or add as test user |

### GitHub OAuth Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `bad_verification_code` | Code expired/used | Retry the flow |
| `incorrect_client_credentials` | Wrong credentials | Verify against OAuth app settings |
| `redirect_uri_mismatch` | Callback URL mismatch | Update in GitHub OAuth app settings |

### Session Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `Invalid session` | Cookie tampered/expired | Clear cookies, retry login |
| `Session not found` | Server restarted (in-memory) | Use Redis for production |

---

## Section 9: Security Best Practices

1. **Never commit secrets** - Add `.env` to `.gitignore`
2. **Use HTTPS in production** - OAuth requires HTTPS for redirect URIs
3. **Rotate secrets regularly** - Generate new client secrets periodically
4. **Limit authorized domains** - Only allow trusted domains
5. **Use short session expiry** - 24 hours recommended
6. **Enable CSRF protection** - Include in auth middleware
7. **Review OAuth app permissions** - Only request necessary scopes

---

## Section 10: References Included

- Google OAuth 2.0 Documentation: https://developers.google.com/identity/protocols/oauth2
- GitHub OAuth Documentation: https://docs.github.com/en/developers/apps/building-oauth-apps
- koa-session Documentation: https://github.com/koajs/session
- OWASP Session Management Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html

---

## Next Steps for Milestone 4

After recreating the oauth-setup.md file, the remaining Milestone 4 work includes:

1. **Session middleware** - koa-session configuration
2. **OAuth routes** - `/auth/login/:provider`, `/auth/callback/:provider`, `/auth/logout`, `/auth/me`
3. **OAuth provider implementations** - Google and GitHub integrations
4. **Auth middleware** - `auth()` function for protected routes
5. **Login UI** - `<scion-login-page>` component
6. **Domain-based authorization** - Check email domain against AUTHORIZED_DOMAINS

---

## Recovery Instructions

To restore the oauth-setup.md file:

1. Copy the content from Sections 1-10 above into a new file at `.design/oauth-setup.md`
2. Or use this report as a reference to recreate the documentation
