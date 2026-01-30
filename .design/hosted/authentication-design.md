# Scion Authentication Design

## Status
**Proposed**

## 1. Overview

This document specifies the authentication mechanisms for Scion's hosted mode. Authentication establishes user identity across multiple client types while maintaining security and usability.

### Authentication Contexts

| Context | Client Type | Auth Method | Token Storage |
|---------|-------------|-------------|---------------|
| Web Dashboard | Browser | OAuth + Session Cookie | HTTP-only cookie |
| CLI (Hub Commands) | Terminal | OAuth + Device Flow | Local file (`~/.scion/credentials.json`) |
| API Direct | Programmatic | API Key or JWT | Client-managed |
| **Development** | Any | Dev Token (Bearer) | Local file (`~/.scion/dev-token`) |

### Goals

1. **Unified Identity** - Single user identity across all client types
2. **Secure Token Management** - Appropriate storage for each context
3. **Developer Experience** - Minimal friction for CLI authentication
4. **Standard Protocols** - OAuth 2.0 / OpenID Connect compliance

### Non-Goals

- Runtime host authentication (addressed in separate design - see Section 9)
- Service-to-service authentication between Hub components
- Multi-tenant Hub federation

---

## 2. Identity Model

### 2.1 User Identity

A user is identified by their email address, which serves as the canonical identifier across OAuth providers.

```go
type User struct {
    ID           string    `json:"id"`           // UUID primary key
    Email        string    `json:"email"`        // Canonical identifier
    DisplayName  string    `json:"displayName"`
    AvatarURL    string    `json:"avatarUrl,omitempty"`

    // OAuth provider info
    Provider     string    `json:"provider"`     // "google", "github", etc.
    ProviderID   string    `json:"providerId"`   // Provider's user ID

    // Status
    Role         string    `json:"role"`         // "admin", "member", "viewer"
    Status       string    `json:"status"`       // "active", "suspended", "pending"

    // Timestamps
    Created      time.Time `json:"created"`
    LastLogin    time.Time `json:"lastLogin"`
}
```

### 2.2 Authentication Tokens

The Hub issues JWT tokens for authenticated sessions:

```go
type TokenClaims struct {
    jwt.RegisteredClaims

    UserID      string   `json:"uid"`
    Email       string   `json:"email"`
    Role        string   `json:"role"`
    TokenType   string   `json:"type"`    // "access", "refresh", "cli"
    ClientType  string   `json:"client"`  // "web", "cli", "api"
}
```

**Token Types:**

| Type | Lifetime | Purpose |
|------|----------|---------|
| `access` | 15 minutes | Short-lived API access |
| `refresh` | 7 days | Token renewal |
| `cli` | 30 days | CLI session (longer-lived for developer convenience) |

---

## 3. Web Authentication (OAuth)

Web authentication uses standard OAuth 2.0 authorization code flow with session cookies.

### 3.1 Flow Diagram

```
┌─────────┐     ┌─────────────┐     ┌──────────────┐     ┌─────────┐
│ Browser │────►│Web Frontend │────►│OAuth Provider│────►│ Hub API │
│         │     │   :9820     │     │(Google/GitHub)│    │ :9810   │
└─────────┘     └─────────────┘     └──────────────┘     └─────────┘
     │                │                    │                  │
     │  1. GET /auth/login/google          │                  │
     │───────────────►│                    │                  │
     │                │  2. Redirect to OAuth                 │
     │◄───────────────│───────────────────►│                  │
     │  3. User authorizes                 │                  │
     │◄───────────────────────────────────►│                  │
     │                │  4. Callback with code                │
     │───────────────►│◄───────────────────│                  │
     │                │  5. Exchange code for tokens          │
     │                │───────────────────►│                  │
     │                │◄───────────────────│                  │
     │                │  6. Create/lookup user                │
     │                │────────────────────────────────────►│
     │                │◄────────────────────────────────────│
     │                │  7. Issue session token               │
     │                │────────────────────────────────────►│
     │                │◄────────────────────────────────────│
     │  8. Set session cookie                                 │
     │◄───────────────│                    │                  │
     │  9. Redirect to app                 │                  │
     │◄───────────────│                    │                  │
```

### 3.2 Session Management

Web sessions use HTTP-only cookies with the following properties:

```typescript
const sessionConfig = {
  name: 'scion:sess',
  maxAge: 24 * 60 * 60 * 1000,  // 24 hours
  httpOnly: true,
  secure: true,                  // HTTPS only in production
  sameSite: 'lax',
  signed: true
};
```

### 3.3 Hub API Endpoints

```
POST /api/v1/auth/login
  Request:  { provider, email, name, avatar, providerToken }
  Response: { user, accessToken, refreshToken }

POST /api/v1/auth/refresh
  Request:  { refreshToken }
  Response: { accessToken, refreshToken }

POST /api/v1/auth/logout
  Request:  { refreshToken? }
  Response: { success: true }

GET /api/v1/auth/me
  Response: { user }
```

---

## 4. CLI Authentication

CLI authentication enables `scion hub` commands to authenticate with a Hub server using a browser-based OAuth flow with localhost callback.

### 4.1 Commands

```bash
# Check authentication status
scion hub auth status

# Authenticate with Hub (opens browser)
scion hub auth login [--hub-url <url>]

# Clear stored credentials
scion hub auth logout
```

### 4.2 Device Authorization Flow

The CLI uses OAuth 2.0 with a localhost redirect for systems with a browser:

```
┌──────────┐     ┌─────────────┐     ┌──────────────┐     ┌─────────┐
│   CLI    │     │  Localhost  │     │OAuth Provider│     │ Hub API │
│ Terminal │     │   :18271    │     │              │     │ :9810   │
└──────────┘     └─────────────┘     └──────────────┘     └─────────┘
     │                 │                    │                  │
     │  1. scion hub auth login            │                  │
     │─────────────────┼───────────────────┼─────────────────►│
     │                 │                   │   2. Get auth URL │
     │◄────────────────┼───────────────────┼──────────────────│
     │  3. Start localhost server          │                  │
     │────────────────►│                   │                  │
     │  4. Open browser with auth URL      │                  │
     │─────────────────┼──────────────────►│                  │
     │                 │  5. User authorizes                  │
     │                 │◄─────────────────►│                  │
     │                 │  6. Redirect to localhost            │
     │                 │◄──────────────────│                  │
     │  7. Receive auth code               │                  │
     │◄────────────────│                   │                  │
     │  8. Exchange code for CLI token     │                  │
     │─────────────────┼───────────────────┼─────────────────►│
     │◄────────────────┼───────────────────┼──────────────────│
     │  9. Store credentials locally       │                  │
     │                 │                   │                  │
```

### 4.3 Implementation Details

#### Localhost Callback Server

```go
// pkg/hub/auth/localhost_server.go

const (
    CallbackPort = 18271  // Arbitrary high port for localhost callback
    CallbackPath = "/callback"
)

type LocalhostAuthServer struct {
    server     *http.Server
    codeChan   chan string
    errChan    chan error
    state      string
}

func (s *LocalhostAuthServer) Start(ctx context.Context) (string, error) {
    // Generate random state for CSRF protection
    s.state = generateRandomState()

    mux := http.NewServeMux()
    mux.HandleFunc(CallbackPath, s.handleCallback)

    s.server = &http.Server{
        Addr:    fmt.Sprintf("127.0.0.1:%d", CallbackPort),
        Handler: mux,
    }

    go s.server.ListenAndServe()

    return fmt.Sprintf("http://127.0.0.1:%d%s", CallbackPort, CallbackPath), nil
}

func (s *LocalhostAuthServer) handleCallback(w http.ResponseWriter, r *http.Request) {
    // Verify state matches
    if r.URL.Query().Get("state") != s.state {
        s.errChan <- fmt.Errorf("state mismatch")
        http.Error(w, "State mismatch", http.StatusBadRequest)
        return
    }

    code := r.URL.Query().Get("code")
    if code == "" {
        errMsg := r.URL.Query().Get("error_description")
        s.errChan <- fmt.Errorf("auth failed: %s", errMsg)
        http.Error(w, "Authentication failed", http.StatusBadRequest)
        return
    }

    // Send success page to browser
    w.Header().Set("Content-Type", "text/html")
    w.Write([]byte(authSuccessHTML))

    s.codeChan <- code
}

func (s *LocalhostAuthServer) WaitForCode(ctx context.Context) (string, error) {
    select {
    case code := <-s.codeChan:
        return code, nil
    case err := <-s.errChan:
        return "", err
    case <-ctx.Done():
        return "", ctx.Err()
    case <-time.After(5 * time.Minute):
        return "", fmt.Errorf("authentication timeout")
    }
}
```

#### CLI Auth Command

```go
// cmd/hub_auth.go

var hubAuthCmd = &cobra.Command{
    Use:   "auth",
    Short: "Manage Hub authentication",
}

var hubAuthLoginCmd = &cobra.Command{
    Use:   "login",
    Short: "Authenticate with Hub server",
    RunE: func(cmd *cobra.Command, args []string) error {
        hubURL, _ := cmd.Flags().GetString("hub-url")
        if hubURL == "" {
            hubURL = config.DefaultHubURL()
        }

        // Start localhost callback server
        authServer := auth.NewLocalhostAuthServer()
        callbackURL, err := authServer.Start(cmd.Context())
        if err != nil {
            return fmt.Errorf("failed to start auth server: %w", err)
        }
        defer authServer.Shutdown()

        // Get OAuth URL from Hub
        client := hub.NewClient(hubURL)
        authURL, err := client.GetAuthURL(cmd.Context(), callbackURL)
        if err != nil {
            return fmt.Errorf("failed to get auth URL: %w", err)
        }

        // Open browser
        fmt.Println("Opening browser for authentication...")
        if err := openBrowser(authURL); err != nil {
            fmt.Printf("Please open this URL in your browser:\n%s\n", authURL)
        }

        // Wait for callback
        fmt.Println("Waiting for authentication...")
        code, err := authServer.WaitForCode(cmd.Context())
        if err != nil {
            return fmt.Errorf("authentication failed: %w", err)
        }

        // Exchange code for token
        token, err := client.ExchangeCode(cmd.Context(), code, callbackURL)
        if err != nil {
            return fmt.Errorf("failed to get token: %w", err)
        }

        // Store credentials
        if err := credentials.Store(hubURL, token); err != nil {
            return fmt.Errorf("failed to store credentials: %w", err)
        }

        fmt.Println("Authentication successful!")
        return nil
    },
}

var hubAuthStatusCmd = &cobra.Command{
    Use:   "status",
    Short: "Show authentication status",
    RunE: func(cmd *cobra.Command, args []string) error {
        hubURL := config.DefaultHubURL()

        creds, err := credentials.Load(hubURL)
        if err != nil {
            fmt.Println("Not authenticated")
            return nil
        }

        // Verify token is still valid
        client := hub.NewClient(hubURL)
        client.SetToken(creds.AccessToken)

        user, err := client.GetCurrentUser(cmd.Context())
        if err != nil {
            fmt.Println("Authentication expired. Run 'scion hub auth login' to re-authenticate.")
            return nil
        }

        fmt.Printf("Authenticated as: %s (%s)\n", user.DisplayName, user.Email)
        fmt.Printf("Hub: %s\n", hubURL)
        return nil
    },
}

var hubAuthLogoutCmd = &cobra.Command{
    Use:   "logout",
    Short: "Clear stored credentials",
    RunE: func(cmd *cobra.Command, args []string) error {
        hubURL := config.DefaultHubURL()

        if err := credentials.Remove(hubURL); err != nil {
            return fmt.Errorf("failed to remove credentials: %w", err)
        }

        fmt.Println("Logged out successfully.")
        return nil
    },
}
```

### 4.4 Credential Storage

CLI credentials are stored in `~/.scion/credentials.json`:

```json
{
  "version": 1,
  "hubs": {
    "https://hub.example.com": {
      "accessToken": "eyJ...",
      "refreshToken": "eyJ...",
      "expiresAt": "2025-02-01T12:00:00Z",
      "user": {
        "id": "user-uuid",
        "email": "user@example.com",
        "displayName": "User Name"
      }
    }
  }
}
```

**Security Considerations:**
- File permissions set to `0600` (owner read/write only)
- Tokens are not encrypted at rest (relies on filesystem permissions)
- Refresh tokens enable automatic token renewal

```go
// pkg/credentials/store.go

const (
    CredentialsFile = "credentials.json"
    FileMode        = 0600
)

type Credentials struct {
    Version int                        `json:"version"`
    Hubs    map[string]*HubCredentials `json:"hubs"`
}

type HubCredentials struct {
    AccessToken  string    `json:"accessToken"`
    RefreshToken string    `json:"refreshToken"`
    ExpiresAt    time.Time `json:"expiresAt"`
    User         *User     `json:"user"`
}

func Store(hubURL string, token *TokenResponse) error {
    path := filepath.Join(config.ScionDir(), CredentialsFile)

    creds, _ := load(path)
    if creds == nil {
        creds = &Credentials{Version: 1, Hubs: make(map[string]*HubCredentials)}
    }

    creds.Hubs[hubURL] = &HubCredentials{
        AccessToken:  token.AccessToken,
        RefreshToken: token.RefreshToken,
        ExpiresAt:    time.Now().Add(token.ExpiresIn),
        User:         token.User,
    }

    data, err := json.MarshalIndent(creds, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(path, data, FileMode)
}

func Load(hubURL string) (*HubCredentials, error) {
    path := filepath.Join(config.ScionDir(), CredentialsFile)
    creds, err := load(path)
    if err != nil {
        return nil, err
    }

    hubCreds, ok := creds.Hubs[hubURL]
    if !ok {
        return nil, ErrNotAuthenticated
    }

    // Check if token needs refresh
    if time.Now().After(hubCreds.ExpiresAt.Add(-5 * time.Minute)) {
        return refreshToken(hubURL, hubCreds)
    }

    return hubCreds, nil
}
```

### 4.5 Headless Authentication

For systems without a browser (CI/CD, remote servers), support API key authentication:

```bash
# Set API key via environment variable
export SCION_API_KEY="sk_live_..."

# Or via config file
scion hub auth set-key <api-key>
```

API keys are created via the web dashboard and stored in the same credentials file.

---

## 5. API Key Authentication

For programmatic access and CI/CD pipelines, users can create API keys.

### 5.1 API Key Format

```
sk_live_<base64-encoded-payload>
```

Payload structure:
```json
{
  "kid": "key-uuid",
  "uid": "user-uuid",
  "created": "2025-01-01T00:00:00Z"
}
```

### 5.2 API Key Management

```
POST /api/v1/auth/api-keys
  Request:  { name, expiresAt?, scopes? }
  Response: { key, keyId, name, createdAt }

GET /api/v1/auth/api-keys
  Response: { keys: [{ keyId, name, lastUsed, createdAt }] }

DELETE /api/v1/auth/api-keys/{keyId}
  Response: { success: true }
```

### 5.3 API Key Usage

API keys are passed via the `Authorization` header:

```
Authorization: Bearer sk_live_...
```

Or via `X-API-Key` header:

```
X-API-Key: sk_live_...
```

---

## 6. Development Authentication (Interim)

> **Status:** Interim solution for development and local testing until full OAuth is implemented.

Development authentication provides a simple, zero-configuration mechanism for local development and testing. It bridges the gap until full OAuth-based authentication is implemented.

### 6.1 Goals

1. **Zero-config local development** - Start the server and immediately use the CLI
2. **Persistent tokens** - Tokens survive server restarts
3. **Environment variable override** - Easy integration with CI/testing
4. **Clear security boundary** - Obvious when running in dev mode
5. **Builds on existing auth** - Uses the standard `Bearer` authentication mechanism

### 6.2 Token Format

```
scion_dev_<32-character-hex-string>
```

Example: `scion_dev_a1b2c3d4e5f6789012345678901234567890abcd`

The `scion_dev_` prefix makes tokens easily identifiable and grep-able in logs.

### 6.3 Server Configuration

```yaml
server:
  auth:
    # Enable development authentication mode
    # WARNING: Not for production use
    devMode: false  # Default: disabled

    # Explicit token (optional)
    # If empty and devMode=true, auto-generate and persist
    devToken: ""

    # Path to token file (optional)
    # Default: ~/.scion/dev-token
    devTokenFile: ""
```

**Environment Variable Mapping:**

| Variable | Maps To |
|----------|---------|
| `SCION_SERVER_AUTH_DEV_MODE` | `server.auth.devMode` |
| `SCION_SERVER_AUTH_DEV_TOKEN` | `server.auth.devToken` |
| `SCION_SERVER_AUTH_DEV_TOKEN_FILE` | `server.auth.devTokenFile` |

### 6.4 Token Resolution Flow

When the server starts with development authentication enabled:

1. Check if a token is explicitly configured (`server.auth.devToken`)
2. If not, check for an existing token file at `~/.scion/dev-token`
3. If no file exists, generate a new cryptographically secure token
4. Store the token in `~/.scion/dev-token` with `0600` permissions
5. Log the token to stdout for easy copy/paste

**Startup Log Output:**
```
Scion Hub API starting on :9810
WARNING: Development authentication enabled - not for production use
Dev token: scion_dev_a1b2c3d4e5f6789012345678901234567890abcd

To authenticate CLI commands, run:
  export SCION_DEV_TOKEN=scion_dev_a1b2c3d4e5f6789012345678901234567890abcd

Or the token has been saved to: ~/.scion/dev-token
```

### 6.5 Client Token Resolution

The client checks for development tokens in the following order:

1. **Explicit option** - `hubclient.WithBearerToken(token)` or `hubclient.WithDevToken(token)`
2. **Environment variable** - `SCION_DEV_TOKEN`
3. **Token file** - `~/.scion/dev-token`

**Client Environment Variables:**

| Variable | Purpose |
|----------|---------|
| `SCION_DEV_TOKEN` | Development token value |
| `SCION_DEV_TOKEN_FILE` | Path to token file (default: `~/.scion/dev-token`) |

### 6.6 Wire Protocol

Development tokens use the standard Bearer authentication scheme:

```http
GET /api/v1/agents HTTP/1.1
Host: localhost:9810
Authorization: Bearer scion_dev_a1b2c3d4e5f6789012345678901234567890abcd
```

This is identical to production Bearer token authentication, ensuring no code path differences between dev and production auth flows.

### 6.7 Implementation

#### Server-Side Token Management

```go
package auth

import (
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

const (
    devTokenPrefix = "scion_dev_"
    devTokenLength = 32 // bytes, results in 64 hex chars
)

// DevAuthConfig holds development authentication settings.
type DevAuthConfig struct {
    Enabled   bool   `koanf:"devMode"`
    Token     string `koanf:"devToken"`
    TokenFile string `koanf:"devTokenFile"`
}

// InitDevAuth initializes development authentication.
// Returns the token to use and any error encountered.
func InitDevAuth(cfg DevAuthConfig, scionDir string) (string, error) {
    if !cfg.Enabled {
        return "", nil
    }

    // Priority 1: Explicit token in config
    if cfg.Token != "" {
        return cfg.Token, nil
    }

    // Determine token file path
    tokenFile := cfg.TokenFile
    if tokenFile == "" {
        tokenFile = filepath.Join(scionDir, "dev-token")
    }

    // Priority 2: Existing token file
    if data, err := os.ReadFile(tokenFile); err == nil {
        token := strings.TrimSpace(string(data))
        if token != "" {
            return token, nil
        }
    }

    // Priority 3: Generate new token
    token, err := generateDevToken()
    if err != nil {
        return "", fmt.Errorf("failed to generate dev token: %w", err)
    }

    // Persist token
    if err := os.WriteFile(tokenFile, []byte(token+"\n"), 0600); err != nil {
        return "", fmt.Errorf("failed to write dev token file: %w", err)
    }

    return token, nil
}

// generateDevToken creates a new cryptographically secure development token.
func generateDevToken() (string, error) {
    bytes := make([]byte, devTokenLength)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return devTokenPrefix + hex.EncodeToString(bytes), nil
}

// IsDevToken returns true if the token appears to be a development token.
func IsDevToken(token string) bool {
    return strings.HasPrefix(token, devTokenPrefix)
}
```

#### Client-Side Token Resolution

```go
package hubclient

import (
    "os"
    "path/filepath"
    "strings"

    "github.com/ptone/scion-agent/pkg/apiclient"
)

// WithDevToken sets a development token for authentication.
func WithDevToken(token string) Option {
    return func(c *client) {
        c.auth = &apiclient.BearerAuth{Token: token}
    }
}

// WithAutoDevAuth attempts to load a development token automatically.
// Checks SCION_DEV_TOKEN env var, then ~/.scion/dev-token file.
func WithAutoDevAuth() Option {
    return func(c *client) {
        token := resolveDevToken()
        if token != "" {
            c.auth = &apiclient.BearerAuth{Token: token}
        }
    }
}

// resolveDevToken finds a development token from environment or file.
func resolveDevToken() string {
    // Priority 1: Environment variable
    if token := os.Getenv("SCION_DEV_TOKEN"); token != "" {
        return token
    }

    // Priority 2: Custom token file from env
    if tokenFile := os.Getenv("SCION_DEV_TOKEN_FILE"); tokenFile != "" {
        if data, err := os.ReadFile(tokenFile); err == nil {
            return strings.TrimSpace(string(data))
        }
    }

    // Priority 3: Default token file
    home, err := os.UserHomeDir()
    if err != nil {
        return ""
    }

    tokenFile := filepath.Join(home, ".scion", "dev-token")
    if data, err := os.ReadFile(tokenFile); err == nil {
        return strings.TrimSpace(string(data))
    }

    return ""
}
```

### 6.8 Usage Examples

#### Starting the Server

```bash
# Start Hub with dev auth (token auto-generated)
scion server start --enable-hub --dev-auth

# Or via config
cat > ~/.scion/server.yaml << EOF
server:
  hub:
    enabled: true
  auth:
    devMode: true
EOF

scion server start --config ~/.scion/server.yaml
```

#### Using the CLI

```bash
# Option 1: Set environment variable (explicit)
export SCION_DEV_TOKEN=scion_dev_a1b2c3d4e5f6789012345678901234567890abcd
scion agent list --hub http://localhost:9810

# Option 2: Automatic (reads from ~/.scion/dev-token)
scion agent list --hub http://localhost:9810

# Option 3: One-liner
SCION_DEV_TOKEN=$(cat ~/.scion/dev-token) scion agent list --hub http://localhost:9810
```

#### CI/Testing Integration

```yaml
# GitHub Actions example
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Start Scion Hub
        run: |
          scion server start --enable-hub --dev-auth --background
          echo "SCION_DEV_TOKEN=$(cat ~/.scion/dev-token)" >> $GITHUB_ENV

      - name: Run integration tests
        run: go test ./integration/...
        env:
          SCION_HUB_URL: http://localhost:9810
          # SCION_DEV_TOKEN already set above
```

### 6.9 Security Constraints

**The server MUST:**

1. Log a clear warning when dev auth is enabled
2. Refuse to start with dev auth if binding to non-localhost AND TLS is disabled
3. Include "dev-mode" in health check responses

```go
func validateDevAuthConfig(cfg *ServerConfig) error {
    if !cfg.Auth.DevMode {
        return nil
    }

    // Warn about dev mode
    log.Warn("Development authentication enabled - not for production use")

    // Block dangerous configurations
    if !cfg.TLS.Enabled && !isLocalhost(cfg.Host) {
        return fmt.Errorf("dev auth requires TLS when binding to non-localhost address")
    }

    return nil
}
```

**Token File Permissions:**
- Token file MUST be created with `0600` permissions (owner read/write only)
- Client SHOULD warn if token file has overly permissive permissions

**Token Entropy:**
- Tokens use 32 bytes (256 bits) of cryptographic randomness
- This provides sufficient entropy to prevent brute-force attacks

**No Token in URLs:**
- Tokens MUST NOT be passed in URL query parameters
- This prevents token leakage in server logs, browser history, and referrer headers

### 6.10 Migration to Production Auth

When OAuth authentication is fully implemented:

1. Dev auth remains available but disabled by default
2. Production deployments set `devMode: false` explicitly
3. The `WithAutoDevAuth()` client option becomes a no-op when `SCION_DEV_TOKEN` is unset and no token file exists
4. Dev tokens are rejected by production servers (check for `scion_dev_` prefix)

---

## 7. Hub API Auth Endpoints

### 7.1 OAuth Initiation (for CLI)

```
GET /api/v1/auth/authorize
  Query: redirect_uri, state
  Response: { authUrl, state }
```

### 7.2 Token Exchange

```
POST /api/v1/auth/token
  Request:  { code, redirectUri, grantType: "authorization_code" }
  Response: { accessToken, refreshToken, expiresIn, user }

POST /api/v1/auth/token
  Request:  { refreshToken, grantType: "refresh_token" }
  Response: { accessToken, refreshToken, expiresIn }
```

### 7.3 Token Validation

```
POST /api/v1/auth/validate
  Request:  { token }
  Response: { valid: true, user, expiresAt }
```

---

## 8. Security Considerations

### 8.1 Token Security

| Aspect | Web | CLI | API Key |
|--------|-----|-----|---------|
| Storage | HTTP-only cookie | Local file (0600) | Local file or env var |
| Transmission | HTTPS only | HTTPS only | HTTPS only |
| Lifetime | 24 hours (session) | 30 days (renewable) | Configurable |
| Revocation | Logout endpoint | Logout command | Dashboard |

### 8.2 PKCE for CLI

CLI authentication uses PKCE (Proof Key for Code Exchange) for additional security:

```go
type PKCEChallenge struct {
    Verifier  string  // Random 43-128 character string
    Challenge string  // SHA256(verifier), base64url encoded
    Method    string  // "S256"
}

func GeneratePKCE() *PKCEChallenge {
    verifier := generateRandomString(64)
    hash := sha256.Sum256([]byte(verifier))
    challenge := base64.RawURLEncoding.EncodeToString(hash[:])

    return &PKCEChallenge{
        Verifier:  verifier,
        Challenge: challenge,
        Method:    "S256",
    }
}
```

### 8.3 Rate Limiting

Authentication endpoints are rate-limited to prevent brute force attacks:

| Endpoint | Limit | Window |
|----------|-------|--------|
| `/auth/login` | 10 | 1 minute |
| `/auth/token` | 20 | 1 minute |
| `/auth/authorize` | 10 | 1 minute |

### 8.4 Audit Logging

All authentication events are logged:

```go
type AuthEvent struct {
    EventType   string    `json:"eventType"`   // login, logout, token_refresh, api_key_created
    UserID      string    `json:"userId"`
    ClientType  string    `json:"clientType"`  // web, cli, api
    IPAddress   string    `json:"ipAddress"`
    UserAgent   string    `json:"userAgent"`
    Success     bool      `json:"success"`
    FailReason  string    `json:"failReason,omitempty"`
    Timestamp   time.Time `json:"timestamp"`
}
```

---

## 9. Future Work: Runtime Host Authentication

> **TODO:** Runtime host authentication will be addressed in a separate design document.

Runtime hosts (Docker, Apple Virtualization, Kubernetes) require a different trust model:

- **Host Registration** - How hosts register with the Hub
- **Host Identity** - Certificates or tokens for host identification
- **Mutual TLS** - Secure communication between Hub and hosts
- **Host Capabilities** - What operations hosts can perform

This is distinct from user authentication and will be designed separately to address the unique security requirements of distributed compute resources.

---

## 10. Implementation Phases

### Phase 0: Development Authentication (Interim)
- [ ] Add `auth.devMode`, `auth.devToken`, `auth.devTokenFile` to config schema
- [ ] Implement `InitDevAuth()` function
- [ ] Add `--dev-auth` flag to `scion server start`
- [ ] Implement `DevAuthMiddleware`
- [ ] Add startup logging for dev token
- [ ] Add validation to block non-localhost + no-TLS + devMode
- [ ] Add `WithDevToken()` option to `hubclient`
- [ ] Add `WithAutoDevAuth()` option to `hubclient`
- [ ] Add `SCION_DEV_TOKEN` environment variable support in CLI

### Phase 1: Web OAuth
- [x] OAuth provider integration (Google, GitHub)
- [x] Session cookie management
- [x] User creation/lookup on login
- [ ] Hub auth endpoints (`/api/v1/auth/*`)

### Phase 2: CLI Authentication
- [ ] `scion hub auth login` command
- [ ] Localhost callback server
- [ ] PKCE implementation
- [ ] Credential storage (`~/.scion/credentials.json`)
- [ ] `scion hub auth status` command
- [ ] `scion hub auth logout` command

### Phase 3: API Keys
- [ ] API key generation endpoint
- [ ] API key validation middleware
- [ ] Key management UI in dashboard
- [ ] `scion hub auth set-key` command

### Phase 4: Security Hardening
- [ ] Rate limiting on auth endpoints
- [ ] Audit logging
- [ ] Token revocation lists
- [ ] Session invalidation on password change

---

## 11. References

- **Permissions System:** `permissions-design.md`
- **Web Frontend:** `web-frontend-design.md`
- **Hub API:** `hub-api.md`
- **OAuth 2.0 RFC:** https://datatracker.ietf.org/doc/html/rfc6749
- **PKCE RFC:** https://datatracker.ietf.org/doc/html/rfc7636
