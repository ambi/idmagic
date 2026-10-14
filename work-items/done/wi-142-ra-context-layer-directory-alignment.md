---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# saml / wsfederation のオーケストレーションを usecases 層へ抽出し層×コンテキスト格子に揃える

`saml` / `wsfederation` に欠けていた `usecases/` 層を新設し、HTTP ハンドラに折り込まれていたブラウザ federation のオーケストレーションを移設した。挙動・HTTP 契約・ドメインイベントは不変。
- `internal/saml/usecases`: SSO 発行判断 `SignInService`（SP 解決・署名検証・認証/再認証ゲート・割当ゲート・claim 発行 → discriminated outcome）と SLO の `LogoutService`（返送先解決・LogoutRequest 検証）。
- `internal/wsfederation/usecases`: passive `SignInService`、`SignOutService`（wreply 検証）、WS-Trust の `WsTrustService`（claim/token type 決定）。WS-Trust active STS の SOAP body 読取・replay・throttle・資格情報検証は HTTP/資格情報固有処理として adapter に残置。
- 依存方向: usecase は adapter/support を import せず、割当ゲートは usecase 内 `ApplicationGate` interface で受け、adapter の `gateAdapter` が `support.ApplicationGate` を値変換で橋渡し（oauth2 `ports.TokenIssuer` パターンに準拠）。assertion / SAMLResponse / RSTR / passive form の構築・署名・直列化は adapter に残置。
- `ARCHITECTURE.md` の Go Package Conventions を更新: per-context `domain/` / `usecases/` の有無は context 固有ロジックの有無で決まり、ドメイン型正本は `internal/shared/spec`（SCL Go binding, ADR-070）。
- 当初起票時の逸脱1（IM/tenancy の `domain/` 欠落）・逸脱4（`bootstrap` 命名）は ADR-070/047 と `CLAUDE.md` の RA 節解釈により違反でないと確認し、実装対象外（重複 ADR も起票せず）。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
