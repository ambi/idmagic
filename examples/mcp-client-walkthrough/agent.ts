// 公式 TypeScript SDK のクライアントだけで MCP ゲートウェイへ接続する Agent の例である。
// IdMagic 固有のコードを持たず、401 の応答から保護リソースのメタデータ（RFC 9728）をたどり、
// 秘密鍵で署名したクライアントアサーション（RFC 7523）で、ゲートウェイ宛てのトークン（RFC 8707）を得る。
import { Client, type FetchLike, PrivateKeyJwtProvider, StreamableHTTPClientTransport } from "@modelcontextprotocol/client";

export type AgentOptions = {
  /** ゲートウェイの公開 URL。IdMagic に登録した接続先の `resource` と同じ値である。 */
  url: string;
  /** IdMagic が Agent の認証用クライアントに発行したクライアント ID。 */
  clientId: string;
  /** Agent の秘密鍵。公開鍵だけを IdMagic へ登録する。 */
  privateKey: JsonWebKey;
  /** 資格情報を送ってよい認可サーバー。IdMagic のテナントの realm の URL である。 */
  issuer: string;
  fetch?: FetchLike;
};

export async function connectAgent(options: AgentOptions): Promise<Client> {
  const provider = new PrivateKeyJwtProvider({
    clientId: options.clientId,
    privateKey: options.privateKey as Record<string, unknown>,
    algorithm: "PS256",
    expectedIssuer: options.issuer,
  });
  const transport = new StreamableHTTPClientTransport(new URL(options.url), {
    authProvider: provider,
    fetch: options.fetch,
  });
  const client = new Client(
    { name: "idmagic-walkthrough-agent", version: "1.0.0" },
    { versionNegotiation: { mode: { pin: "2026-07-28" } } },
  );
  await client.connect(transport);
  return client;
}

if (import.meta.main) {
  const [tool, documentId] = process.argv.slice(2);
  const required = (name: string) => {
    const value = process.env[name];
    if (!value) throw new Error(`${name} を設定してください。`);
    return value;
  };
  const client = await connectAgent({
    url: required("MCP_URL"),
    clientId: required("AGENT_CLIENT_ID"),
    privateKey: JSON.parse(await Bun.file(required("AGENT_PRIVATE_JWK")).text()),
    issuer: required("IDMAGIC_REALM"),
  });
  try {
    if (!tool) {
      console.log((await client.listTools()).tools.map((t) => t.name).join("\n"));
    } else {
      console.log(JSON.stringify(await client.callTool({ name: tool, arguments: { document_id: documentId ?? "handbook" } })));
    }
  } finally {
    await client.close();
  }
}
