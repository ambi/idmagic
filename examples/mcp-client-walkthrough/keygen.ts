// Agent の鍵の組を作る。秘密鍵は Agent の手元に置き、公開鍵だけを IdMagic へ登録する。
// 単独で起動すると、秘密鍵の JWK をファイルへ書き、公開鍵の JWK を標準出力へ書く。
export type AgentKey = { privateJwk: JsonWebKey; publicJwk: JsonWebKey };

/** IdMagic のクライアントアサーションが受け入れる PS256 の RSA の鍵を作る。 */
export async function generateAgentKey(): Promise<AgentKey> {
  const pair = (await crypto.subtle.generateKey(
    { name: "RSA-PSS", modulusLength: 2048, publicExponent: new Uint8Array([1, 0, 1]), hash: "SHA-256" },
    true,
    ["sign", "verify"],
  )) as CryptoKeyPair;
  const privateJwk = { ...(await crypto.subtle.exportKey("jwk", pair.privateKey)), alg: "PS256" };
  const { kty, n, e } = await crypto.subtle.exportKey("jwk", pair.publicKey);
  return { privateJwk, publicJwk: { kty, n, e, alg: "PS256", use: "sig" } };
}

if (import.meta.main) {
  const out = process.argv[2] ?? "agent-private.jwk.json";
  const key = await generateAgentKey();
  await Bun.write(out, JSON.stringify(key.privateJwk));
  console.error(`秘密鍵を ${out} に書きました。このファイルは Agent の外へ出しません。`);
  console.log(JSON.stringify(key.publicJwk));
}
