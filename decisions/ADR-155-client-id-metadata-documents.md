---
status: accepted
authors: [claude]
created_at: 2026-08-08
---

# ADR-155: Client ID Metadata Documents (CIMD) を非永続の代替クライアント解決経路として採用する

## コンテキスト

[[wi-341-oauth-client-id-metadata-document-cimd]]。MCP authorization 仕様 2026-07-28 改訂は
クライアント登録の優先順位を「pre-registration → Client ID Metadata Documents (CIMD) →
Dynamic Client Registration (RFC 7591、fallback) → 手動入力」とし、DCR を非推奨にした
([[ADR-055]] の対象改訂 pin 更新時の監査で判明)。CIMD は `client_id` を client が自己ホストする
HTTPS URL にし、authorization server がその URL から `redirect_uris` 等を都度 fetch する。
根拠仕様は IETF `draft-ietf-oauth-client-id-metadata-document-**00**`(2026年8月時点でごく初期、
未 RFC 化)。client 指定の任意 URL を server が fetch する新しい外部入力面が加わるため、
永続化するかどうか・SSRF 対策・MVP でどこまで受理するかを決める必要がある。

## 決定

1. **対象 draft のバージョンを `-00` に固定する。** [[ADR-055]] と同じ運用: draft が改訂されたら
   本 ADR を更新し pin し直す。
2. **CIMD 解決結果は永続化しない。** `client_id` が `https` scheme かつ非空 path を持つ URL 形状で、
   既存の登録済み `OAuth2Client` 検索(`OAuth2ClientRepository.FindByID`)がヒットしない場合に限り、
   その URL から CIMD ドキュメントを fetch して都度解決する。解決結果は `OAuth2Client` と同じ形に
   マップし、`/authorize`・`/par`・`/token` の既存ロジック(redirect_uri 照合・PKCE・consent 表示・
   scope 検証)をそのまま再利用する。`Application` 割当ゲート(`ApplicationGate`)は現在
   DCR 自己登録クライアントにも Application レコードを要求していない(未登録 = 許可、
   [[wi-341-oauth-client-id-metadata-document-cimd]] 実装調査で確認)ため、CIMD もこの既存挙動に揃える。
3. **fetch は SSRF-safe な専用 fetcher で行い、既存の `shared/security/tokens_jose` の
   JWKS 解決 (`JWKResolver`) と同じ安全境界(https 限定・DNS 解決後の public IP 限定・
   リダイレクト上限・タイムアウト・応答サイズ上限)を共通化して再利用する。** キャッシュは
   HTTP cache header を解釈せず固定 5 分 TTL とする(`CIMD00-CACHE` は `adoption: partial`)。
   header 準拠のキャッシュ制御は本 ADR の対象改訂が要求する SHOULD だが、実装複雑度に対し
   優先度が低いと判断し見送った。
4. **MVP は `token_endpoint_auth_method` が未指定または `none` のドキュメントのみ受理する。**
   それ以外の値を宣言するドキュメントは fail-closed で拒否する(`private_key_jwt` 等の
   CIMD 経由対応は別途 wi で扱う、`adoption: excluded`)。`client_id` フィールドは fetch した
   URL と厳密一致必須、`redirect_uris` は必須かつ非空。
5. **`scope` はドキュメントの自己申告値をそのまま採用する**(未指定時は `openid` のみ)。
   これは RFC 7591 DCR の `ClientRegistrationRequest.scope` が既に自己申告(admin 側の
   カタログ照合なし)であるのと同じ信頼モデルで、CIMD 専用の新しい scope ポリシーを
   別途設けない。
6. **Discovery (`/.well-known/oauth-authorization-server`) に
   `client_id_metadata_document_supported: true` を追加する。**

現在の設計は [`backend/oauth2/ARCHITECTURE.md`](../backend/oauth2/ARCHITECTURE.md) にある。

## 却下した代替案

- **CIMD 解決結果を `OAuth2Client` として永続化する**: ドキュメントの「client 側で
  redirect_uris 等を再登録なしに更新できる」という CIMD の利点を失い、DCR と重複する
  レジストリ管理コストが増える。ドキュメント自体を正とするため非永続を選んだ。
- **MVP で `private_key_jwt` も CIMD 経由で対応する**: CIMD ドキュメントが指す `jwks_uri` への
  さらなる外部 fetch が連鎖し検証すべき SSRF 経路が増える。`none` のみに絞り、
  別 wi へ切り出す。
- **CIMD クライアント専用の scope カタログ(tenant 全体の `scopes_supported` からの
  導出・admin 側の許可リスト等)を新設する**: 既存 DCR の自己申告モデルと非対称になり
  複雑さが増す。DCR と揃えることで新しいポリシーを増やさない。

## 影響

- 新規 standards エントリ (`draft-ietf-oauth-client-id-metadata-document`)、
  `DiscoveryDocument.client_id_metadata_document_supported`、`Authorize` /
  `PushAuthorizationRequest` の CIMD 解決に関する記述、新規 event を `spec/contexts/oauth2.yaml`
  へ追加する。
- 新規 Go adapter (`backend/oauth2/client/cimd_http`)、SSRF-safe fetcher の共通化
  (`shared/security/tokens_jose` からの抽出)。
