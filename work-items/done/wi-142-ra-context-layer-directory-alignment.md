---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-08
---

# saml / wsfederation のオーケストレーションを usecases 層へ抽出し層×コンテキスト格子に揃える

## Motivation

`REGENERATIVE_ARCHITECTURE.md §3.8` は「ディレクトリ構造はドメイン・仕様の構造をそのまま表現し反映し叫ばなければならない」とし、各 context 内部で層を繰り返す規範レイアウト（`domain/` `ports/` `usecases/` `adapters/`）を定める。`ARCHITECTURE.md` の "Go Package Conventions" と ADR-047（層×コンテキスト格子）/ ADR-070（technical shared context）も同じ格子を確立している。

現状 `internal/<context>/` の層ディレクトリ有無を棚卸ししたところ、`saml` と `wsfederation` だけが `usecases/` を持たず、SSO/SLO/sign-in の**オーケストレーション（SP/RP 解決・署名検証・fail-closed 割当ゲート・claim 発行・assertion 構築・イベント発行）が `adapters/http` ハンドラに折り込まれている**。これは他の protocol context（oauth2 等）が持つ `usecases/` 層を欠く不整合であり、1 機能を読むのに HTTP 境界処理とアプリケーション論理が混在したハンドラを読む必要が生じている。

## 調査結果（現状の層マトリクスと ADR 照合）

`internal/<context>/` の層ディレクトリ有無:

| context | domain | usecases | ports | adapters |
| --- | --- | --- | --- | --- |
| application | ✓ | ✓ | ✓ | ✓ |
| authentication | ✓ | ✓ | ✓ | ✓ |
| identitymanagement | ✗ | ✓ | ✓ | ✓ |
| oauth2 | ✓ | ✓ | ✓ | ✓ |
| saml | ✓ | **✗** | ✓ | ✓ |
| scim | ✓ | ✓ | ✓ | ✓ |
| tenancy | ✗ | ✓ | ✓ | ✓ |
| wsfederation | ✓ | **✗** | ✓ | ✓ |

既存 ADR と照合した結果、当初「逸脱」と見えたもののうち **2 件は違反ではなく決定済みの設計**だった:

- **`domain/` 欠落（identitymanagement / tenancy）は違反ではない。** ADR-070 §2 が「`internal/shared/spec` は SCL Go binding であり複数 context が参照する共通の内側」と決定済み。ドメイン型（`User` `Group` `Agent` 等）の正本は `shared/spec` で、per-context の `domain/` は *context 固有のドメインロジック*（saml の AuthnRequest 解析、wsfed の claims mapping 等）がある場合にのみ置く。IM/tenancy はそれを持たないため `domain/` が無いのが正しい。
- **`internal/bootstrap` の命名は違反ではない。** RA §3.8 の `infrastructure/` は section-addressable な例示（`CLAUDE.md` 明記）。ADR-068→070 で `infrastructure` の語義は横断アダプタ実装へ再割当され、Layer 6（main / DI / seed）は `internal/bootstrap` として多数の ADR が参照する確立した規約。

したがって本 WI の**真に actionable な核は `saml` / `wsfederation` の `usecases/` 欠落（上表の唯一の不整合）**である。

## Scope

- 機能変更・挙動変更なし（SCL セクションの変更なし）。純粋な構造リファクタリングと第 2 層（`ARCHITECTURE.md`）の同期。
- `internal/saml/`: `adapters/http` の SSO/SLO オーケストレーションを `internal/saml/usecases/` へ抽出。assertion / response / logout-response の構築・署名は saml `ports` の builder 抽象を介す（oauth2 の `ports.TokenIssuer` パターンに倣い、usecase が adapter を import しない依存方向を守る）。
- `internal/wsfederation/`: `adapters/http` の passive sign-in / sign-out / WS-Trust オーケストレーションを `internal/wsfederation/usecases/` へ抽出。同上の builder 抽象を用いる。
- `adapters/http` ハンドラは wire 変換・HTTP status・cookie・CSP・redirect などの境界処理と、header からの authn 解決・request からの client IP 抽出に縮約する。
- `ARCHITECTURE.md` の "Go Package Conventions" を更新し、per-context の `domain/` / `usecases/` の有無が「context 固有ロジックの有無」で決まること（`shared/spec` = SCL binding、ADR-070/047 準拠）を明記する。

## Out of Scope

- `domain/` 欠落（IM / tenancy）と `bootstrap` 命名の変更。上述のとおり ADR-070/047 で決定済みで違反ではない。ADR 起票も不要（既存 ADR の重複になる）。
- `internal/shared/spec` の domain 型を per-context へ分割する移設。
- 共有 `support.ApplicationGate` / `support.Authenticator` そのものの再配置（saml/wsfed usecase 側は自前の interface で受け、adapter が既存 support 実装を橋渡しする）。
- `internal/tenancy/context.go` の再配置。リクエストコンテキストのアクセサで実害が小さく、今回の核から外す。
- SCL（`spec/scl.yaml` / `spec/contexts/*.yaml`）の意味変更、HTTP ルート・wire contract・DB スキーマの変更。

## Plan

- **技術方針**: 挙動不変のリファクタリング。既存の `saml_handler_test.go` / `wsfed_handler_test.go`（HTTP 経由の E2E 的テスト）を回帰ネットとして緑を保つ。
- **依存方向**: oauth2 の確立パターンに倣い、usecase は adapter/support を import しない。
  - assertion / SAMLResponse / RSTR / LogoutResponse の構築・署名は各 context の `ports` に builder interface を定義し、`adapters` 側（`samltoken` / `samlresponse` / `wsfed` + `FederationSigner`）が実装する。
  - 割当ゲート（`support.ApplicationGate.EvaluateApplicationAccess`）と authn 解決は usecase パッケージ内の小さな interface で受け、adapter が既存 support 実装を橋渡しする（decision 型は usecase 側に定義し adapter で写す）。
- **usecase 出力**: SSO/sign-in は discriminated outcome（`NeedLogin` / `Rejected` / `Forbidden` / `Issued`）を返し、adapter が HTTP へ写す。ドメインイベントの発行点・内容は現状と一致させる。
- **段階**: (a) saml 抽出 → verify → (b) wsfederation 抽出 → verify → (c) `ARCHITECTURE.md` 同期 → (d) 全体 verify。

## Tasks

- [x] T001 [App] `internal/saml`: SSO/SLO のオーケストレーションを `internal/saml/usecases/`（`SignInService` / `LogoutService`）へ抽出し、割当ゲートを usecase の interface で受けて adapter が `support.ApplicationGate` を橋渡し、ハンドラを wire/HTTP 境界に縮約した。
- [x] T002 [Verify] `just build-go` 通過、`just test-go` で saml 全パッケージ green。
- [x] T003 [App] `internal/wsfederation`: passive sign-in（`SignInService`）・sign-out（`SignOutService`）・WS-Trust トークン発行判断（`WsTrustService`）を `internal/wsfederation/usecases/` へ抽出。WS-Trust の SOAP body 読取・replay・throttle・資格情報検証は active STS 固有の HTTP/資格情報処理として adapter に残し、claim/token type 決定のみ usecase 化した。
- [x] T004 [Verify] `just build-go` 通過、`just test-go` で wsfederation 全パッケージ green。
- [x] T005 [Arch] `ARCHITECTURE.md` の Go Package Conventions を更新し、per-context の `domain/` / `usecases/` の有無が「context 固有ロジックの有無」で決まること（`shared/spec` = SCL binding、ADR-070/047 準拠）と、usecase が adapter を import しない依存方向を明記した。
- [x] T006 [Verify] `just verify-go`（lint + race）、`just yaml-check` / `just yaml-check-work-items` / `just check-ids` を通した。

## Verification

- `just verify-go`（build / lint / race test）。リファクタリングにつき既存テストが緑のまま維持されることを確認。
- `just yaml-check-work-items` と `just check-ids`。
- 目視: `internal/saml/` と `internal/wsfederation/` が `domain` / `usecases` / `ports` / `adapters` の 4 層を持ち、`ARCHITECTURE.md` の記述と一致すること。

## Risk Notes

- **medium**: saml/wsfederation の抽出は SAML/WS-Fed の署名検証・fail-closed 割当ゲート・open-redirect 防止といったセキュリティ上デリケートな経路に触れる。ただし挙動不変で、HTTP 経由の既存ハンドラテストが振る舞いの回帰ネットになる。
- イベント発行の内容・タイミング（`SamlSignInRejected` / `WsFedSignInIssued` 等）を現状と厳密に一致させることに注意する。

## Completion

- **Completed At**: 2026-07-08
- **Summary**:
  `saml` / `wsfederation` に欠けていた `usecases/` 層を新設し、HTTP ハンドラに折り込まれていたブラウザ federation のオーケストレーションを移設した。挙動・HTTP 契約・ドメインイベントは不変。
  - `internal/saml/usecases`: SSO 発行判断 `SignInService`（SP 解決・署名検証・認証/再認証ゲート・割当ゲート・claim 発行 → discriminated outcome）と SLO の `LogoutService`（返送先解決・LogoutRequest 検証）。
  - `internal/wsfederation/usecases`: passive `SignInService`、`SignOutService`（wreply 検証）、WS-Trust の `WsTrustService`（claim/token type 決定）。WS-Trust active STS の SOAP body 読取・replay・throttle・資格情報検証は HTTP/資格情報固有処理として adapter に残置。
  - 依存方向: usecase は adapter/support を import せず、割当ゲートは usecase 内 `ApplicationGate` interface で受け、adapter の `gateAdapter` が `support.ApplicationGate` を値変換で橋渡し（oauth2 `ports.TokenIssuer` パターンに準拠）。assertion / SAMLResponse / RSTR / passive form の構築・署名・直列化は adapter に残置。
  - `ARCHITECTURE.md` の Go Package Conventions を更新: per-context `domain/` / `usecases/` の有無は context 固有ロジックの有無で決まり、ドメイン型正本は `internal/shared/spec`（SCL Go binding, ADR-070）。
  - 当初起票時の逸脱1（IM/tenancy の `domain/` 欠落）・逸脱4（`bootstrap` 命名）は ADR-070/047 と `CLAUDE.md` の RA 節解釈により違反でないと確認し、実装対象外（重複 ADR も起票せず）。
- **Verification Results**:
  - `just build-go` - passed
  - `just verify-go`（golangci-lint + race test）- passed
  - `just yaml-check` / `just yaml-check-work-items`（142 files OK）- passed
  - `just check-ids`（223 ids OK）- passed
  - saml / wsfederation の既存 HTTP ハンドラテスト（`saml_handler_test.go` 488 行 / `wsfed_handler_test.go` 545 行）が無改修で green。挙動・イベント発行の回帰なしを確認。
- **Affected Guarantees State**: 新規・変更なし。SCL の規範振る舞いおよび保証義務は不変（純構造リファクタリング）。
