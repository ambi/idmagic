import http from 'k6/http'
import crypto from 'k6/crypto'
import encoding from 'k6/encoding'
import { check, fail } from 'k6'
import { Trend } from 'k6/metrics'

const baseURL = __ENV.IDMAGIC_BASE_URL || 'http://host.docker.internal:8080/realms/default'
const browserOrigin = __ENV.IDMAGIC_BROWSER_ORIGIN || 'http://localhost:8080'
const clientID = __ENV.IDMAGIC_CLIENT_ID || '00000000-0000-4000-8000-000000000021'
const clientSecret = __ENV.IDMAGIC_CLIENT_SECRET || 'demo-client-secret'
const username = __ENV.IDMAGIC_USERNAME || 'alice'
const password = __ENV.IDMAGIC_PASSWORD || 'demo-password-1234'
const redirectURI = __ENV.IDMAGIC_REDIRECT_URI || 'http://localhost:3000/callback'
const tokenLatency = new Trend('idmagic_token_latency', true)

export const options = {
  vus: Number(__ENV.K6_VUS || 1),
  iterations: Number(__ENV.K6_ITERATIONS || 1),
  thresholds: {
    checks: ['rate>0.99'],
    http_req_failed: ['rate<0.001'],
    idmagic_token_latency: ['p(99)<300'], // OAuth2/objective/TokenLatency
  },
}

function form(values) {
  return Object.entries(values)
    .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(value)}`)
    .join('&')
}

function pkce() {
  // The smoke test checks the end-to-end PKCE contract. A deterministic verifier
  // is sufficient here because each VU owns a distinct authorization transaction.
  const verifier = `k6-${__VU}-${__ITER}-0123456789012345678901234567890123456789`
  const digest = crypto.sha256(verifier, 'binary')
  return { verifier, challenge: encoding.b64encode(digest, 'rawurl') }
}

function basicParams() {
  return {
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded',
      Authorization: `Basic ${encoding.b64encode(`${clientID}:${clientSecret}`)}`,
    },
  }
}

function issueToken(values) {
  const response = http.post(`${baseURL}/token`, form(values), basicParams())
  tokenLatency.add(response.timings.duration)
  check(response, { 'token response is 200': (r) => r.status === 200 })
  if (response.status !== 200) {
    fail(`token request failed: HTTP ${response.status} ${response.body}`)
  }
  return response.json()
}

export function setup() {
  const healthURL = baseURL.replace(/\/realms\/[^/]+$/, '')
  const response = http.get(`${healthURL}/health`)
  if (!check(response, { 'target is healthy': (r) => r.status === 200 })) {
    fail(`IdMagic is unavailable at ${baseURL}`)
  }
}

export default function () {
  const { verifier, challenge } = pkce()
  const state = `k6-${__VU}-${__ITER}`
  const authorize = http.get(`${baseURL}/authorize?${form({
    response_type: 'code', client_id: clientID, redirect_uri: redirectURI,
    scope: 'openid offline_access', state, nonce: state,
    code_challenge: challenge, code_challenge_method: 'S256',
  })}`, { redirects: 0 })
  check(authorize, { 'authorization starts a login transaction': (r) => r.status === 302 || r.status === 303 })
  if (authorize.status !== 302 && authorize.status !== 303) {
    fail(`authorization did not start a login transaction: HTTP ${authorize.status} ${authorize.body}`)
  }

  const transaction = http.get(`${baseURL}/api/auth/transaction`)
  const csrfToken = transaction.json('csrf_token')
  const login = http.post(`${baseURL}/api/auth/login`, JSON.stringify({ username, password }), {
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken, Origin: browserOrigin },
  })
  check(login, { 'login succeeds': (r) => r.status === 200 })
  if (login.status !== 200) {
    fail(`login failed: HTTP ${login.status} ${login.body}`)
  }
  let redirectTo = login.json('redirect_to')
  if (login.json('next') === '/consent') {
    const consentTransaction = http.get(`${baseURL}/api/auth/transaction`)
    const consent = http.post(`${baseURL}/api/auth/consent`, JSON.stringify({ action: 'allow' }), {
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': consentTransaction.json('csrf_token'), Origin: browserOrigin },
    })
    check(consent, { 'consent succeeds': (r) => r.status === 200 })
    redirectTo = consent.json('redirect_to')
  }
  const code = /[?&]code=([^&]+)/.exec(redirectTo || '')?.[1]
  check(code, { 'authorization code is returned': (value) => value !== null })

  const authorizationCode = issueToken({ grant_type: 'authorization_code', code, code_verifier: verifier, redirect_uri: redirectURI })
  const refreshed = issueToken({ grant_type: 'refresh_token', refresh_token: authorizationCode.refresh_token })
  check(refreshed, { 'refresh rotation returns a replacement token': (value) => Boolean(value.refresh_token) })
  const clientCredentials = issueToken({ grant_type: 'client_credentials', scope: 'openid' })
  check(clientCredentials, { 'client credentials returns an access token': (value) => Boolean(value.access_token) })
}
