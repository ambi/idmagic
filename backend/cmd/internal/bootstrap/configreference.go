package bootstrap

import (
	"fmt"
	"slices"
	"strings"
)

// configReferenceSection is one process-scoped group of the generated
// ConfigurationReference. Load calls exactly the Load*Config functions that
// process runs at startup, so the section lists what that process really
// reads rather than a hand-kept list beside it (REQ-SYSTEM-017).
type configReferenceSection struct {
	Title     string
	Processes string
	Summary   string
	Load      func(*ConfigLoader)
}

var configReferenceSections = []configReferenceSection{
	{
		Title:     "Shared",
		Processes: "idmagic, idmagic-worker, idmagic-batch, idmagic-seed",
		Summary:   "Persistence, notification, WebAuthn, authorization, and key-custody settings every process assembles its adapters from.",
		Load:      func(l *ConfigLoader) { LoadSharedConfig(l) },
	},
	{
		Title:     "API",
		Processes: "idmagic",
		Summary:   "HTTP listener, hardening, security headers, and endpoint rate limits.",
		Load:      func(l *ConfigLoader) { LoadAPIConfig(l) },
	},
	{
		Title:     "Worker",
		Processes: "idmagic-worker",
		Summary:   "Durable job queue lanes, runner cadence, and the resident sweep loops.",
		Load:      func(l *ConfigLoader) { LoadWorkerConfig(l) },
	},
	{
		Title:     "Seed",
		Processes: "idmagic (startup seed), idmagic-seed",
		Summary:   "The explicit seed request. idmagic applies it once at startup when SEED_PROFILE is set; idmagic-seed uses it as the default for its own flags.",
		Load:      func(l *ConfigLoader) { LoadSeedConfig(l) },
	},
}

// configFieldDescriptions is the one hand-written part of the
// ConfigurationReference: what a key means. Every other column is recorded
// by the loader itself. A key with no entry here, and an entry naming a key
// no process reads, both fail RenderConfigReference and its test, so this
// table cannot drift into a stale parallel list of settings.
var configFieldDescriptions = map[string]string{
	// Shared
	"PERSISTENCE":                  "Storage backend. `postgres` requires DATABASE_URL.",
	"OBSERVABILITY":                "Set to `otel` to export OTLP traces and metrics. Pull-based /metrics is always served regardless of this setting.",
	"AUTHZEN":                      "Authorization decision point. `remote` delegates to an AuthZEN PDP and requires AUTHZEN_URL.",
	"AUTHZEN_URL":                  "AuthZEN policy decision point endpoint. Required when AUTHZEN=remote.",
	"WEBAUTHN_RP_ID":               "WebAuthn relying-party ID, e.g. `localhost`. WebAuthn and passkeys stay disabled while it is unset.",
	"WEBAUTHN_RP_ORIGINS":          "Browser origins allowed to run WebAuthn ceremonies, e.g. `http://localhost:5173`. Required when WEBAUTHN_RP_ID is set.",
	"WEBAUTHN_RP_DISPLAY_NAME":     "Relying-party display name shown by authenticators.",
	"DATABASE_URL":                 "PostgreSQL connection string. Required when PERSISTENCE=postgres.",
	"DB_MAX_CONNS":                 "Upper bound on the PostgreSQL connection pool.",
	"DB_MIN_CONNS":                 "Connections the pool keeps warm.",
	"DB_MAX_CONN_IDLE_TIME":        "Idle time after which a pooled connection is closed.",
	"DB_MAX_CONN_LIFETIME":         "Maximum lifetime of a pooled connection.",
	"DB_CONNECT_TIMEOUT":           "Deadline for establishing a PostgreSQL connection.",
	"DB_QUERY_TIMEOUT":             "Deadline for a single query, held until its results are fully read.",
	"DB_BREAKER_FAILURE_THRESHOLD": "Failure ratio at which the PostgreSQL circuit breaker opens.",
	"DB_BREAKER_COOLDOWN":          "How long the PostgreSQL circuit breaker stays open before probing again.",
	"DB_BREAKER_MIN_REQUESTS":      "Requests observed in a window before the failure ratio is allowed to open the breaker.",
	"KEY_PROVIDER":                 "Signing key custody. `vault` requires VAULT_ADDR and VAULT_TOKEN; unset keeps keys in the application database.",
	"VAULT_ADDR":                   "Vault base address. Required when KEY_PROVIDER=vault.",
	"VAULT_TOKEN":                  "Vault token. Required when KEY_PROVIDER=vault.",
	"VAULT_TRANSIT_MOUNT":          "Vault Transit engine mount path used for signing keys.",
	"VAULT_KEY_PREFIX":             "Prefix for per-tenant Vault Transit signing key names.",
	"DATA_KEY_PROVIDER":            "Master-key custody for envelope-encrypted reversible secrets. `openbao` requires OPENBAO_ADDR and OPENBAO_TOKEN; unset uses an in-process cleartext keyset (development only).",
	"OPENBAO_ADDR":                 "OpenBao base address. Required when DATA_KEY_PROVIDER=openbao.",
	"OPENBAO_TOKEN":                "OpenBao token. Required when DATA_KEY_PROVIDER=openbao.",
	"OPENBAO_TRANSIT_MOUNT":        "OpenBao Transit engine mount path.",
	"OPENBAO_DATA_KEY_PREFIX":      "Prefix for per-tenant OpenBao Transit key names (`{prefix}/{tenant_id}`).",
	"EMAIL_SENDER":                 "Delivery channel for password reset and notification email. `smtp` requires SMTP_HOST and SMTP_FROM.",
	"SMTP_HOST":                    "SMTP server hostname. Required when EMAIL_SENDER=smtp.",
	"SMTP_FROM":                    "From address on outgoing mail. Required when EMAIL_SENDER=smtp.",
	"SMTP_TLS":                     "SMTP transport security.",
	"SMTP_PORT":                    "SMTP port. `0` selects the default port for the chosen SMTP_TLS mode (587 starttls, 465 implicit, 25 none).",
	"SMTP_USERNAME":                "SMTP username. Leave unset for an unauthenticated relay.",
	"SMTP_PASSWORD":                "SMTP password.",
	"SMTP_HELO":                    "HELO/EHLO name announced to the SMTP server.",
	"SMTP_TIMEOUT_SECONDS":         "Deadline for one SMTP delivery attempt.",
	"DEFAULT_LOCALE":               "Last-resort language for notification email. An unsupported value fails startup.",
	"BREACHED_PASSWORD_CHECKER":    "Breached-password check performed on password changes. `hibp` calls the Have I Been Pwned range API.",

	// API
	"ISSUER":                     "Public base URL this deployment issues tokens under. It must match what relying parties resolve.",
	"ADDR":                       "Listen address of the HTTP server.",
	"OTEL_SERVICE_NAME":          "service.name reported on logs, metrics, and traces.",
	"LOG_LEVEL":                  "Minimum severity written to stdout.",
	"REQUEST_ID_TRUST_INBOUND":   "Reuse an inbound X-Request-ID instead of generating one. Enable only behind a proxy that owns and sanitizes the header.",
	"TENANT_BASE_DOMAIN":         "Parent hostname enabling the subdomain endpoint style at `{realm}.<domain>`, e.g. `id.example.com`. Unset keeps every tenant on path routing.",
	"TRUSTED_FORWARDED_HOPS":     "Number of trusted X-Forwarded-For hops in front of this process, used to resolve the real client IP for the login throttle and the endpoint rate limits. `0` distrusts the header entirely.",
	"DRAIN_GRACE_PERIOD_SECONDS": "Seconds to keep serving after SIGTERM before shutting the listener down (idmagic), or to let in-flight jobs finish (idmagic-worker).",
	"PAGINATION_CURSOR_SECRET":   "HMAC secret signing keyset pagination cursors. Set it explicitly in any multi-replica deployment: otherwise each replica generates its own at startup and rejects cursors issued by another.",

	"HTTP_READ_HEADER_TIMEOUT": "Deadline for reading request headers.",
	"HTTP_READ_TIMEOUT":        "Deadline for reading a whole request.",
	"HTTP_WRITE_TIMEOUT":       "Deadline for writing a response.",
	"HTTP_IDLE_TIMEOUT":        "How long an idle keep-alive connection is held open.",
	"HTTP_MAX_BODY_BYTES":      "Request body limit; a larger body is rejected with 413.",

	"CSP_REPORT_ONLY":         "Send the Content Security Policy as Content-Security-Policy-Report-Only for a staged rollout.",
	"CSP_REPORT_URI":          "report-uri collecting Content Security Policy violations.",
	"HSTS_ENABLED":            "Emit Strict-Transport-Security. Off by default because the TLS terminator owns it.",
	"HSTS_MAX_AGE_SECONDS":    "max-age of the Strict-Transport-Security header when enabled.",
	"HSTS_INCLUDE_SUBDOMAINS": "Add includeSubDomains to Strict-Transport-Security.",

	"RATE_LIMIT_TOKEN_MAX_REQUESTS":                        "`/token` fixed-window limit, keyed by client_id and IP.",
	"RATE_LIMIT_TOKEN_WINDOW_SECONDS":                      "Window length for the `/token` limit.",
	"RATE_LIMIT_AUTHORIZE_MAX_REQUESTS":                    "`/authorize` fixed-window limit, keyed by IP and client_id.",
	"RATE_LIMIT_AUTHORIZE_WINDOW_SECONDS":                  "Window length for the `/authorize` limit.",
	"RATE_LIMIT_PAR_MAX_REQUESTS":                          "`/par` fixed-window limit, keyed by IP and client_id.",
	"RATE_LIMIT_PAR_WINDOW_SECONDS":                        "Window length for the `/par` limit.",
	"RATE_LIMIT_DEVICE_AUTHORIZATION_MAX_REQUESTS":         "`/device_authorization` fixed-window limit, keyed by client_id and IP.",
	"RATE_LIMIT_DEVICE_AUTHORIZATION_WINDOW_SECONDS":       "Window length for the `/device_authorization` limit.",
	"RATE_LIMIT_BACKCHANNEL_AUTHENTICATION_MAX_REQUESTS":   "`/bc-authorize` fixed-window limit, keyed by client_id and IP.",
	"RATE_LIMIT_BACKCHANNEL_AUTHENTICATION_WINDOW_SECONDS": "Window length for the `/bc-authorize` limit.",
	"RATE_LIMIT_PASSWORD_RESET_MAX_REQUESTS":               "`/api/auth/forgot_password` fixed-window limit, keyed by the submitted identifier and IP.",
	"RATE_LIMIT_PASSWORD_RESET_WINDOW_SECONDS":             "Window length for the password reset limit.",
	"RATE_LIMIT_LOGIN_MAX_REQUESTS":                        "`/api/auth/login` fixed-window limit, keyed by IP. Separate from, and in addition to, the per-account login throttle.",
	"RATE_LIMIT_LOGIN_WINDOW_SECONDS":                      "Window length for the login limit.",

	// Worker
	"WORKER_ID":              "Identifies this worker in job leases. Defaults to the hostname, then to a generated id.",
	"JOB_WORKER_LANES":       "Execution lanes this process claims. The default keeps one process on every lane; a per-lane production deployment sets exactly one.",
	"JOB_WORKER_CONCURRENCY": "Jobs executed in parallel per lane, unless the lane overrides it below.",

	"JOB_WORKER_CONCURRENCY_LATENCY_SENSITIVE": "Overrides JOB_WORKER_CONCURRENCY for the latency_sensitive lane.",
	"JOB_WORKER_CONCURRENCY_DEFAULT":           "Overrides JOB_WORKER_CONCURRENCY for the default lane.",
	"JOB_WORKER_CONCURRENCY_BULK":              "Overrides JOB_WORKER_CONCURRENCY for the bulk lane.",
	"JOB_POLL_INTERVAL":                        "How often a runner polls its lane for claimable jobs.",
	"JOB_LEASE_DURATION":                       "Lease held on a claimed job. Another worker reclaims the job once it expires.",
	"JOB_BACKOFF_BASE":                         "First retry delay after a job attempt fails.",
	"JOB_BACKOFF_CAP":                          "Upper bound on the exponential retry delay.",
	"EPHEMERAL_SWEEP_INTERVAL":                 "How often expired rows in short-TTL ephemeral stores are reclaimed.",
	"SHARED_SIGNALS_DELIVERY_INTERVAL":         "How often due outbound Security Event Tokens are delivered.",

	// Seed
	"SEED_PROFILE":                   "Explicit seed profile applied at startup. Unset means no seeding.",
	"SEED_ENVIRONMENT":               "Environment the seed applies to. Required when SEED_PROFILE is set.",
	"SEED_MANIFEST":                  "Root seed manifest path. Defaults to the profile's own manifest.",
	"SEED_GENERATOR_SEED":            "Deterministic generator seed for the performance profile.",
	"SEED_SECRET_ROOT":               "Root directory for relative `file` secret locators in a manifest.",
	"SEED_FIRST_PARTY_REDIRECT_URIS": "Redirect URIs for the seeded first-party clients. Required for a production bootstrap.",
}

// externallyOwnedConfigNote documents the variables idmagic responds to
// without reading them itself, so an operator reading the reference is not
// left thinking they do not exist.
const externallyOwnedConfigNote = `## Externally owned variables

These are read by libraries rather than by idmagic, so they are not part of the validated
configuration above and a malformed value does not fail startup:

| Variable | Owner | Purpose |
| --- | --- | --- |
| ` + "`OTEL_EXPORTER_OTLP_ENDPOINT`" + ` | OpenTelemetry SDK | OTLP/HTTP collector endpoint used when OBSERVABILITY=otel. |
| ` + "`VITE_DEFAULT_LOCALE`" + ` | Frontend build | Startup default UI locale baked into the React build. |
| ` + "`VITE_DEMO_LOGIN_ENABLED`" + ` | Frontend build | Shows the demo login shortcut outside the Vite dev server. |
`

// RenderConfigReference renders CONFIGURATION.md from the fields each
// process's Load*Config records, so the reference is generated from the same
// code that parses and validates the values (REQ-SYSTEM-017). Secret fields
// render as `secret` with no default: their values never leave the process.
func RenderConfigReference() (string, error) {
	var out strings.Builder
	out.WriteString("<!-- Generated by `mise run generate-config-reference`. Do not edit by hand. -->\n\n")
	out.WriteString("# Configuration Reference\n\n")
	out.WriteString("Every environment variable idmagic reads at startup, grouped by the processes that read it.\n")
	out.WriteString("Values are validated before any process opens a listener, connects to a dependency, or applies a\n")
	out.WriteString("seed: a missing required value, a malformed or out-of-range value, and a conflicting combination\n")
	out.WriteString("all abort startup with every problem reported at once.\n\n")
	out.WriteString("A `secret` value is never written to an error message, a log line, or this reference.\n")

	documented := map[string]bool{}
	var missing []string
	for _, section := range configReferenceSections {
		l := NewConfigLoader(func(string) string { return "" })
		section.Load(l)

		fmt.Fprintf(&out, "\n## %s\n\n", section.Title)
		fmt.Fprintf(&out, "Read by: %s\n\n", section.Processes)
		fmt.Fprintf(&out, "%s\n\n", section.Summary)
		out.WriteString("| Variable | Type | Default | Required | Purpose |\n| --- | --- | --- | --- | --- |\n")

		for _, field := range l.Fields() {
			documented[field.Key] = true
			description, ok := configFieldDescriptions[field.Key]
			if !ok {
				missing = append(missing, field.Key)
				continue
			}
			fmt.Fprintf(&out, "| `%s` | %s | %s | %s | %s |\n",
				field.Key, fieldTypeColumn(field), fieldDefaultColumn(field), requiredColumn(field), description)
		}
	}

	if len(missing) > 0 {
		slices.Sort(missing)
		return "", fmt.Errorf("configFieldDescriptions has no entry for %s", strings.Join(missing, ", "))
	}
	if orphans := undocumentedDescriptions(documented); len(orphans) > 0 {
		return "", fmt.Errorf("configFieldDescriptions describes %s, which no process reads", strings.Join(orphans, ", "))
	}

	out.WriteString("\n")
	out.WriteString(externallyOwnedConfigNote)
	return out.String(), nil
}

func undocumentedDescriptions(read map[string]bool) []string {
	var orphans []string
	for key := range configFieldDescriptions {
		if !read[key] {
			orphans = append(orphans, key)
		}
	}
	slices.Sort(orphans)
	return orphans
}

func fieldTypeColumn(field ConfigField) string {
	column := field.Type
	if len(field.Allowed) > 0 {
		column += ": " + "`" + strings.Join(field.Allowed, "`, `") + "`"
	} else if field.Constraint != "" {
		column += " (" + field.Constraint + ")"
	}
	return column
}

func fieldDefaultColumn(field ConfigField) string {
	if field.Secret || field.Default == "" {
		return "—"
	}
	return "`" + field.Default + "`"
}

func requiredColumn(field ConfigField) string {
	if field.Required {
		return "yes"
	}
	if field.RequiredWhen != "" {
		return "when `" + field.RequiredWhen + "`"
	}
	return "no"
}
