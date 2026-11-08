// 手順書の操作を、実在のクライアント、改修していない MCP サーバー、idmagic-mcp-gateway、IdMagic で順に再現する。
// IdMagic へは公開の管理 API とプロトコルのエンドポイントだけを使う。
// 使い方：IDMAGIC_BIN と MCP_GATEWAY_BIN に実行ファイルを指定して `bun run walkthrough.ts` を実行する。
import { createHash, randomBytes } from "node:crypto";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import type { Client } from "@modelcontextprotocol/client";
import { connectAgent } from "./agent.ts";
import { generateAgentKey } from "./keygen.ts";
import { startDocumentServer } from "./server.ts";

const adminConsoleClientID = "00000000-0000-4000-8000-000000000022";
const adminUser = "alice";
const adminPassword = "demo-password-1234";

type Check = { step: string; expected: string; observed: string; ok: boolean };
const checks: Check[] = [];
function check(step: string, expected: string, observed: string, ok: boolean) {
  checks.push({ step, expected, observed, ok });
  console.log(`${ok ? "ok  " : "FAIL"} ${step}: ${observed}`);
}

const cleanups: (() => unknown)[] = [];
const workDir = mkdtempSync(join(tmpdir(), "idmagic-mcp-walkthrough-"));

function required(name: string): string {
  const value = process.env[name];
  if (!value) throw new Error(`${name} を設定してください。`);
  return value;
}

/** 空いているポートを一つ得る。得た直後に閉じるので、すぐに使う。 */
function freePort(): number {
  const listener = Bun.listen({ hostname: "127.0.0.1", port: 0, socket: { data() {} } });
  const port = listener.port;
  listener.stop(true);
  return port;
}

/** localhost の自己署名の証明書を作る。この試験の TLS の前段だけで使う。 */
async function selfSignedCertificate(): Promise<{ cert: string; key: string; certFile: string }> {
  const certFile = join(workDir, "localhost.pem");
  const keyFile = join(workDir, "localhost-key.pem");
  const openssl = Bun.spawn(
    ["openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes", "-days", "1", "-subj", "/CN=localhost",
      "-addext", "subjectAltName=DNS:localhost,IP:127.0.0.1", "-keyout", keyFile, "-out", certFile],
    { stdout: "ignore", stderr: "pipe" },
  );
  if ((await openssl.exited) !== 0) throw new Error(`openssl が失敗しました: ${await new Response(openssl.stderr).text()}`);
  return { cert: await Bun.file(certFile).text(), key: await Bun.file(keyFile).text(), certFile };
}

/** 平文の HTTP の待ち受けの前に、TLS を終端する前段を置く。本番では利用者のリバースプロキシが担う。 */
function tlsFront(tls: { cert: string; key: string }, target: string) {
  const server = Bun.serve({
    hostname: "127.0.0.1",
    port: 0,
    tls,
    idleTimeout: 0,
    fetch(request) {
      const url = new URL(request.url);
      const headers = new Headers(request.headers);
      headers.set("X-Forwarded-Proto", "https");
      return fetch(target + url.pathname + url.search, {
        method: request.method,
        headers,
        body: request.body,
        redirect: "manual",
      }).catch(() => new Response("upstream unavailable", { status: 502 }));
    },
  });
  cleanups.push(() => server.stop(true));
  return server;
}

function spawnLogged(name: string, command: string[], env: Record<string, string>) {
  const logFile = join(workDir, `${name}.log`);
  const child = Bun.spawn(command, { env: { ...process.env, ...env }, stdout: Bun.file(logFile), stderr: Bun.file(`${logFile}.err`) });
  cleanups.push(() => child.kill());
  return { child, logFile };
}

async function waitFor<T>(what: string, probe: () => Promise<T | undefined>, timeoutMs = 60_000): Promise<T> {
  const deadline = Date.now() + timeoutMs;
  let lastError: unknown;
  while (Date.now() < deadline) {
    try {
      const value = await probe();
      if (value !== undefined) return value;
    } catch (error) {
      lastError = error;
    }
    await Bun.sleep(250);
  }
  throw new Error(`${what} を待ち切れませんでした: ${lastError ?? "timeout"}`);
}

type Recorded = { method: string; url: string; status: number; mcpMethod?: string };

/** 試験の CA を信頼し、クライアントの HTTP の往復を記録する fetch である。 */
function recordingFetch(ca: string, log: Recorded[]) {
  return async (input: string | URL | Request, init?: RequestInit): Promise<Response> => {
    const response = await fetch(input, { ...init, tls: { ca } } as RequestInit);
    const url = input instanceof Request ? input.url : String(input);
    let mcpMethod: string | undefined;
    if (typeof init?.body === "string" && init.body.startsWith("{")) {
      try {
        mcpMethod = JSON.parse(init.body).method;
      } catch {}
    }
    log.push({ method: init?.method ?? "GET", url, status: response.status, mcpMethod });
    return response;
  };
}

// --- 管理者の操作 -------------------------------------------------------------

type Jar = Map<string, string>;
function storeCookies(jar: Jar, response: Response) {
  for (const line of response.headers.getSetCookie()) {
    const [pair] = line.split(";");
    const index = pair.indexOf("=");
    jar.set(pair.slice(0, index).trim(), pair.slice(index + 1).trim());
  }
}
const cookieHeader = (jar: Jar) => [...jar].map(([k, v]) => `${k}=${v}`).join("; ");

/** 管理画面と同じ認可コードフローで管理者としてログインし、管理 API のトークンを得る。 */
async function loginAsAdmin(realm: string, ca: string): Promise<string> {
  const jar: Jar = new Map();
  const call = async (url: string, init: RequestInit = {}) => {
    const headers = new Headers(init.headers);
    if (jar.size) headers.set("Cookie", cookieHeader(jar));
    const response = await fetch(url, { ...init, headers, redirect: "manual", tls: { ca } } as RequestInit);
    storeCookies(jar, response);
    return response;
  };
  const redirectURI = `${realm}/callback`;
  const verifier = randomBytes(32).toString("base64url");
  const challenge = createHash("sha256").update(verifier).digest("base64url");
  const follow = async (start: string): Promise<string | undefined> => {
    let next = start;
    for (let i = 0; i < 10; i++) {
      if (next.startsWith(redirectURI)) return new URL(next).searchParams.get("code") ?? undefined;
      const response = await call(next);
      const location = response.headers.get("Location");
      if (!location) return undefined;
      next = new URL(location, next).toString();
    }
    throw new Error("リダイレクトが多すぎます。");
  };
  const authorize = `${realm}/authorize?${new URLSearchParams({
    response_type: "code", client_id: adminConsoleClientID, redirect_uri: redirectURI,
    scope: "openid profile idmagic.admin", state: randomBytes(16).toString("base64url"),
    code_challenge: challenge, code_challenge_method: "S256",
  })}`;
  await follow(authorize);
  const transaction = (await (await call(`${realm}/api/auth/transaction`)).json()) as { csrf_token: string };
  const login = await call(`${realm}/api/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Origin: new URL(realm).origin, "X-Csrf-Token": transaction.csrf_token },
    body: JSON.stringify({ username: adminUser, password: adminPassword }),
  });
  if (login.status !== 200) throw new Error(`ログインできません: HTTP ${login.status} ${await login.text()}`);
  const { redirect_to } = (await login.json()) as { redirect_to: string };
  const code = await follow(new URL(redirect_to, realm).toString());
  if (!code) throw new Error("認可コードを得られません。");
  const token = await call(`${realm}/token`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({ grant_type: "authorization_code", code, redirect_uri: redirectURI, client_id: adminConsoleClientID, code_verifier: verifier }),
  });
  const body = (await token.json()) as { access_token?: string };
  if (!body.access_token) throw new Error(`管理 API のトークンを得られません: ${JSON.stringify(body)}`);
  return body.access_token;
}

function adminAPI(realm: string, ca: string, token: string) {
  return async <T = unknown>(method: string, path: string, body?: unknown): Promise<T> => {
    const response = await fetch(`${realm}${path}`, {
      method,
      headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
      body: body === undefined ? undefined : JSON.stringify(body),
      tls: { ca },
    } as RequestInit);
    const text = await response.text();
    if (response.status >= 300) throw new Error(`${method} ${path}: HTTP ${response.status} ${text}`);
    return (text ? JSON.parse(text) : undefined) as T;
  };
}

// --- 試験の本体 ---------------------------------------------------------------

async function outcome(run: () => Promise<unknown>): Promise<string> {
  try {
    const result = (await run()) as { isError?: boolean; content?: { text?: string }[] };
    return result?.isError ? `tool error: ${result.content?.[0]?.text}` : "ok";
  } catch (error) {
    const e = error as { name?: string; code?: unknown; status?: unknown; message?: string };
    return `${e.name ?? "Error"}: ${e.message ?? String(error)}`;
  }
}

async function main() {
  const idmagicBin = required("IDMAGIC_BIN");
  const gatewayBin = required("MCP_GATEWAY_BIN");
  const tls = await selfSignedCertificate();

  // 1. IdMagic を https の発行者で起動する。
  const apiPort = freePort();
  const idmagicFront = tlsFront(tls, `http://127.0.0.1:${apiPort}`);
  const origin = `https://localhost:${idmagicFront.port}`;
  const realm = `${origin}/realms/default`;
  const idmagic = spawnLogged("idmagic", [idmagicBin], {
    ADDR: `127.0.0.1:${apiPort}`, ISSUER: origin, PERSISTENCE: "memory",
    SEED_ENVIRONMENT: "development", SEED_PROFILE: "development",
    SEED_FIRST_PARTY_REDIRECT_URIS: `${realm}/callback`,
    DEMO_USER_PASSWORD: adminPassword, DEMO_CLIENT_SECRET: randomBytes(16).toString("hex"),
    WEBAUTHN_RP_ID: "localhost", WEBAUTHN_RP_ORIGINS: origin, WEBAUTHN_RP_DISPLAY_NAME: "IdMagic walkthrough",
  });
  await waitFor("IdMagic の起動", async () => {
    if (idmagic.child.exitCode !== null) throw new Error(`IdMagic が終了しました（${idmagic.logFile}.err）`);
    const response = await fetch(`${realm}/.well-known/openid-configuration`, { tls: { ca: tls.cert } } as RequestInit);
    return response.ok ? true : undefined;
  }, 120_000);

  // 2. 管理者としてログインする。以降の設定は管理画面と同じ管理 API で行う。
  const admin = adminAPI(realm, tls.cert, await loginAsAdmin(realm, tls.cert));

  // 3. 改修していない MCP サーバーを起動する。
  const upstream = startDocumentServer();
  cleanups.push(() => upstream.stop());

  // 4. 接続先を登録し、評価用のクライアントでゲートウェイを起動する。
  const gatewayPort = freePort();
  const gatewayFront = tlsFront(tls, `http://127.0.0.1:${gatewayPort}`);
  const resource = `https://localhost:${gatewayFront.port}/mcp`;
  const evaluator = await admin<{ client_id: string; client_secret: string }>("POST", "/api/admin/v1/applications", { name: "文書 MCP Gateway", type: "service" });
  const server = await admin<{ id: string }>("POST", "/api/admin/v1/mcp-resource-servers", { resource, name: "文書 MCP", client_id: evaluator.client_id });
  const secretFile = join(workDir, "gateway-secret");
  writeFileSync(secretFile, `${evaluator.client_secret}\n`, { mode: 0o600 });
  const configFile = join(workDir, "gateway.yaml");
  writeFileSync(configFile, [
    `listen: 127.0.0.1:${gatewayPort}`,
    `resource: ${resource}`,
    `upstream: ${upstream.url}`,
    "idmagic:",
    `  realm_url: ${realm}`,
    `  ca_file: ${tls.certFile}`,
    `  client_id: "${evaluator.client_id}"`,
    `  client_secret_file: ${secretFile}`,
    "mapping:",
    "  catalog: true",
    "",
  ].join("\n"), { mode: 0o600 });
  const gateway = spawnLogged("gateway", [gatewayBin, "--config", configFile], {});

  // 5. Gateway が送ったカタログのツールを公開する。
  const catalog = await waitFor("Gateway のツールカタログ", async () => {
    if (gateway.child.exitCode !== null) throw new Error(`ゲートウェイが終了しました（${gateway.logFile}.err）`);
    const body = await admin<{ tools?: { name: string; title?: string }[] }>("GET", `/api/admin/v1/mcp-resource-servers/${server.id}/tool-catalog`).catch(() => undefined);
    return body?.tools && body.tools.length >= 4 ? body.tools : undefined;
  });
  check("ツールカタログの送信", "上流の 4 つのツールが IdMagic へカタログとして送られる", catalog.map((t) => t.name).sort().join(", "), catalog.length === 4);
  const rule = (name: string, extra: Record<string, unknown> = {}) => ({
    resource_type: "idmagic_mcp_operation", relation: "invoke", scopes: ["mcp"],
    tool: { name, display_name: catalog.find((t) => t.name === name)?.title ?? name }, ...extra,
  });
  const published = await admin<{ access_rules: { tool: { name: string; assignment_resource_id: string } }[] }>("PATCH", `/api/admin/v1/mcp-resource-servers/${server.id}`, {
    name: "文書 MCP",
    access_rules: [
      rule("read_document"),
      rule("write_document", { condition: { deny_when: 'token.principal_type == "agent"' } }),
      rule("delete_document"),
      rule("publish_document", { requires_approved_operation: true, authorization_detail_type: "document_publication" }),
    ],
  });
  const assignment = (name: string) => published.access_rules.find((r) => r.tool.name === name)?.tool.assignment_resource_id ?? "";

  // 6. Agent を登録し、公開鍵で認証するクライアントを紐付け、閲覧、更新、公開を割り当てる。削除は割り当てない。
  const { privateJwk, publicJwk } = await generateAgentKey();
  const agent = await admin<{ id: string }>("POST", "/api/admin/v1/agents", { name: "文書アシスタント", kind: "autonomous" });
  const agentClient = await admin<{ client: { client_id: string } }>("POST", "/api/admin/v1/clients", {
    client_name: "文書アシスタント", client_type: "confidential", redirect_uris: [], grant_types: ["client_credentials"],
    token_endpoint_auth_method: "private_key_jwt", scope: "mcp", jwks: { keys: [publicJwk] }, fapi_profile: "none",
  });
  await admin("POST", `/api/admin/v1/agents/${agent.id}/credentials`, { client_id: agentClient.client.client_id });
  const tuple = (name: string) => ({ resource_type: "idmagic_mcp_operation", resource_id: assignment(name), relation: "invoke", subject_type: "agent", subject_id: agent.id });
  await admin("POST", "/api/admin/v1/authorization/relation-tuples", { writes: ["read_document", "write_document", "publish_document"].map(tuple) });

  // 7. 実在のクライアントで接続する。
  const exchanges: Recorded[] = [];
  const tokenRequests = () => exchanges.filter((x) => x.url.endsWith("/token")).length;
  let client: Client;
  {
    client = await connectAgent({
      url: resource, clientId: agentClient.client.client_id, privateKey: privateJwk, issuer: realm,
      fetch: recordingFetch(tls.cert, exchanges),
    });
    cleanups.unshift(() => client.close());
    const first = exchanges.find((x) => x.url === resource);
    const discovered = exchanges.some((x) => x.url.includes("/.well-known/oauth-protected-resource"));
    check("認可の入口", "最初の要求は 401 になり、クライアントが保護リソースのメタデータをたどってトークンを得る",
      `first=${first?.status}, metadata=${discovered}, token requests=${tokenRequests()}`, first?.status === 401 && discovered && tokenRequests() === 1);
  }
  const tools = await outcome(() => client.listTools());
  check("ツールの一覧", "Agent 自身の資格で得たトークンで tools/list が通る", tools, tools === "ok");

  const executed = () => upstream.calls.length;
  // reissue が false の段階では、クライアントがトークンを取り直さないことも確かめる。
  // 撤回と拒否は既発行のトークンのまま次の呼び出しから効くことを、取り直しで見誤らないためである。
  const callAndObserve = async (step: string, tool: string, wantOk: boolean, expected: string, reissue = false) => {
    const before = executed();
    const tokensBefore = tokenRequests();
    const result = await outcome(() => client.callTool({ name: tool, arguments: { document_id: "handbook" } }));
    const reached = executed() > before;
    const reissued = tokenRequests() - tokensBefore;
    const last = [...exchanges].reverse().find((x) => x.url === resource);
    check(step, expected, `${result.split("\n")[0]}; HTTP ${last?.status}; upstream=${reached}; new token requests=${reissued}`,
      (result === "ok") === wantOk && reached === wantOk && (reissue || reissued === 0));
    return last?.status;
  };

  await callAndObserve("許可", "read_document", true, "割り当てた閲覧が上流で実行される");
  const forbidden = await callAndObserve("割当のない操作", "delete_document", false, "割り当てていない削除は 403 で止まり、上流へ届かない");
  // T004：403 に insufficient_scope がないので、SDK は認可をやり直さない。
  check("403 への応答", "クライアントはトークンを取り直さず、エラーとして返す", `HTTP ${forbidden}`, forbidden === 403);
  await callAndObserve("条件の拒否", "write_document", false, "割り当てていても、条件が真なら 403 で止まり、上流へ届かない");
  const approval = await callAndObserve("承認が必要な操作", "publish_document", false, "人の承認が必要な操作は安全側で止まり、上流へ届かない");
  check("承認が必要な操作の応答", "承認した操作をゲートウェイが扱えないため 503 になる（wi-83492）", `HTTP ${approval}`, approval === 503);

  // 8. 割当を撤回する。同じトークンのまま、次の呼び出しから拒否される。
  await admin("POST", "/api/admin/v1/authorization/relation-tuples", { deletes: [tuple("read_document")] });
  await callAndObserve("撤回", "read_document", false, "割当を取り消すと、トークンを取り直さずに次の呼び出しから 403 になる");
  await admin("POST", "/api/admin/v1/authorization/relation-tuples", { writes: [tuple("read_document")] });
  await callAndObserve("割当の再付与", "read_document", true, "割当を戻すと再び実行される");

  // 9. Agent を停止する。手元のトークンも新しい発行も止まる。
  await admin("POST", `/api/admin/v1/agents/${agent.id}/disable`);
  const tokensBefore = tokenRequests();
  await callAndObserve("停止", "read_document", false, "無効にした Agent のトークンは拒否され、取り直しも拒否される", true);
  const reissue = exchanges.filter((x) => x.url.endsWith("/token")).slice(tokensBefore);
  check("停止後の発行", "クライアントは 401 を受けて取り直しを試み、IdMagic が拒否する", reissue.map((x) => `token HTTP ${x.status}`).join(", ") || "none",
    reissue.length > 0 && reissue.every((x) => x.status >= 400));
}

let failed = false;
try {
  await main();
} catch (error) {
  failed = true;
  console.error(error);
  console.error(`ログ: ${workDir}`);
} finally {
  for (const cleanup of cleanups.reverse()) {
    try {
      await cleanup();
    } catch {}
  }
}
const failures = checks.filter((c) => !c.ok);
console.log(`\n${checks.length - failures.length}/${checks.length} checks passed`);
for (const f of failures) console.log(`FAIL ${f.step}: expected ${f.expected}; observed ${f.observed}`);
if (failed || failures.length > 0) {
  console.log(`ログ: ${workDir}`);
  process.exit(1);
}
rmSync(workDir, { recursive: true, force: true });
