---
status: in_progress
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-18
priority: p3
depends_on: [wi-21934-decide-module-dependency-order-and-break-cycles]
change_kind: refactor
affected_spec:
  - { path: spec/modules/oauth2/models.tsp, symbol: IdMagic.Contract.ConsentNotFoundError }
  - { path: spec/modules/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.RevokeMyConsent }
  - { path: spec/modules/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.GetAdminConsent }
  - { path: spec/modules/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.RevokeAdminConsent }
  - { path: docs/modules/oauth2/features/consent.md, requirement: REQ-OAUTH2-032, impact: conforms }
---

# 本人の同意管理の経路を OAuth2 へ移す

## 動機

[依存の向きの決定](../done/wi-21934-decide-module-dependency-order-and-break-cycles.md)で、`Authentication → OAuth2` を除く向きを選んだ。
公開パッケージの利用でもモジュールの循環は残るため、この組を独立して解消する。

本人の同意一覧と取り消しの操作は、TypeSpec では OAuth2 の操作、要件では REQ-OAUTH2-032 と REQ-OAUTH2-002 であり、実装だけが Authentication にある。
この置き場所のずれが逆向きの import を生んでいる。

また、本人と管理者の同意の取り消しと管理者の同意の取得は、要件どおり 404 と `consent_not_found` を返すが、TypeSpec はこの応答を宣言していない。

## 対象範囲

- `ListMyConsents` と `RevokeMyConsent` のハンドラーを、Authentication から OAuth2 の HTTP アダプターへ移す。
- Authentication から同意操作の依存を除き、`Authentication → OAuth2` の本番 import をなくす。
- `RevokeMyConsent`、`GetAdminConsent`、`RevokeAdminConsent` の TypeSpec に 404 `consent_not_found` を宣言する。
- 実装後に循環を再計測し、消えた違反 ID だけを台帳から除き、依存の図を生成し直す。

## 対象外

- 同意取り消しのクォータ修正（[wi-23517](wi-23517-release-the-consent-quota-when-the-user-revokes-a-consent.md)）、認証再開の変更。
- IdManagement が使う OAuth2 の `AccountConsents` の扱い（[wi-62674](wi-62674-isolate-identity-oauth-lifecycle-and-account-queries.md)）。
- 同意以外の操作の TypeSpec の欠落。
- ほかの組の循環の解消。

## 設計

### 境界の判断の記録

入力の参照：[依存の向きを選ぶ基準](../../docs/design/application/backend.md#依存の向きを選ぶ基準)の「本人のアカウント画面からの同意の管理」の行、[依存の向きと差し替え点](../../docs/design/application/design-guidelines.md#依存の向きと差し替え点)。

着手時の再調査では、Authentication から OAuth2 への本番 import は次の 3 箇所であり、すべて本人の同意操作に由来する。

| 箇所 | 使うもの |
| --- | --- |
| `backend/authentication/wiring.go` | `RouteDeps.Consents` の型 `consentports.AccountConsents` |
| `backend/authentication/internal/deps_http/deps.go` | `Deps.Consents` の型 |
| `backend/authentication/internal/handlers_http/account_consents_handler.go` | `oauthdomain.Consent`、`ConsentState`、`consentports.ErrConsentNotFound` |

OAuth2 はすでに `/api/account/v1/approval-requests` を自分の経路として登録している。
本人の解決は共有の `support.Authenticator` の `ResolveAuthentication`、ブラウザー要求の検証は同じ `VerifyBrowserRequest`、表示名は `ClientDisplayNameResolver` で行い、どれも Authentication と同じ値が組み立て地点から渡る。

| 候補 | 制約への適合 | 変更シナリオの波及 | 採否 |
| --- | --- | --- | --- |
| A. ハンドラーを OAuth2 へ移し、OAuth2 の同意の usecase を直接呼ぶ | 操作の所有者と実装の置き場所が一致し、逆向きの import が消える。新しい型、ポート、変換は要らない | 同意の表現の変更は OAuth2 の中で閉じる | 採用 |
| B. Authentication に同意操作のポートと値を定義し、組み立て地点で OAuth2 へ変換する | import は消えるが、OAuth2 の `AccountConsents` と同じ操作の契約、値、エラーを複製する | 同意の表現を変えるたびに二つの契約と変換を直す | 不採用 |
| C. 現状を維持する | 循環が残る | 同意の表現の変更が Authentication へ波及する | 不採用 |

起票時の設計は、A を「ルーターと本人認証の責務まで動かす」として退けていた。
再調査では、ルーターの group と本人認証の認証器は両モジュールで共有され、どちらも動かす必要はなかった。

移す処理と保つ応答：

- 一覧は、本人の sub を解決し、`ListConsentsForSub` の結果を表示名とともに `no-store` で返す。
- 取り消しは、ブラウザー要求を検証してから本人の sub を解決し、`RevokeConsent(ctx, deps, sub, sub, client_id, now)` を呼び、204 と `no-store` を返す。対象 sub は本人認証の結果に固定し、URL からは client_id だけを受け取る。
- 未認証は、Authentication の `WriteAccountError` と同じ `support.WriteAccessTokenError` で書く。
- 同意がなければ、管理側と同じ `writeConsentError` で 404 と `consent_not_found` を返す。
- 同意の状態遷移と `ConsentRevoked` の発行は OAuth2 の usecase が引き続き行う。

### 仕様の差分

要件文は変えない。REQ-OAUTH2-032 と REQ-OAUTH2-038 がすでに 404 と `consent_not_found` を定めている。
TypeSpec に `ConsentNotFoundError` を加え、`RevokeMyConsent`、`GetAdminConsent`、`RevokeAdminConsent` の応答に 404 を宣言する。
本人の取り消しの 404 は HTTP 越しに検査されていなかったため、移設の前に REQ-OAUTH2-032 を引くテストを足す。

## 計画

着手時に既存の仕様、TypeSpec、名指した実装とテストを読み、操作列ごとの成功、拒否、途中失敗と再実行を照合する。
新しい型と操作のシグネチャ、変換の担当、必須依存と確定の境界をコードの前に記す。
この結果が上の保証維持と両立しない場合は、実装へ進む前に親項目の判断を見直す。
保証の追加や変更が必要になれば別の仕様変更として扱う。

## タスク

- [x] T001 [Design] import の全入口と作用を再調査し、境界の型と操作、変換、確定順を設計へ記す。
- [x] T002 [App] 同意の一覧と取り消しの経路を OAuth2 へ移し、Authentication の同意の依存を除いて逆向きの辺を解消する。
- [x] T003 [Docs] 公開範囲と現在の設計を同期し、実測で消えた負債だけを削除して依存図を再生成する。
- [x] T004 [Verify] 既存の保証、正式な配線と境界を検査し、仕様差分を確認する。

## 検証

- `mise run test-go-package -- ./backend/authentication/internal/handlers_http` の `TestHandleListAccountConsentsReturnsGrantedConsentsOnly`、`TestHandleRevokeAccountConsentRequiresBrowserVerification`、`TestHandleRevokeAccountConsentSucceeds` と `account_consent_refusal_effects_test.go` を起点にし、OAuth2 の同意操作と正式な配線は `mise run test-go-changed` で確認する。
- `mise run lint-go`：変更した Go の静的検査を行う。
- `mise run check-boundaries` と `mise run check-boundary-debt-ratchet`：辺の解消、公開範囲と負債の非増加を確認する。
- `mise run check-contract-drift`、`mise run check-status-drift` と `mise run check-published-api-compat`：HTTP の表現と応答を変えていないことを確認する。
- `mise run check-work-items`、`mise run check-links`、`mise run spec-diff`：記録と参照、規範への影響を確認する。

テストを変更しない構造変更は既存テストの成功を証拠にする。
保証を表すテストが欠けている場合は、公開状態に応じた特性化または要件のテストを実装より先に用意し、その段階の証拠と変更耐性の手順を選ぶ。

## リスク

import を消しただけで実行時の再入と必須の作用を失わないよう、変換前後の公開経路と各確定点を比較する。
必須の作用を省略できる未配線経路を増やさず、既存の構築条件とテナント境界を保つ。

## 要約

TypeSpec に `ConsentNotFoundError` を加え、`RevokeMyConsent`、`GetAdminConsent`、`RevokeAdminConsent` の 404 を宣言した。
要件文は変えていない。REQ-OAUTH2-032 と REQ-OAUTH2-038 がすでに定めていた応答を契約へ写した。

本人の同意の一覧と取り消しのハンドラーを Authentication から OAuth2 の HTTP アダプターへ移し、OAuth2 の同意の usecase を直接呼ぶようにした。
Authentication からは同意操作と表示名の解決の依存が消え、`Authentication → OAuth2` の循環が実測で消えた。台帳は 14 件から 13 件になった。
起票時の設計は要求側のポートと変換を置く案だったが、同じ操作の契約、値、エラーを複製するため採らなかった。
ルーターの group と本人認証の認証器は両モジュールで共有されており、移設で動かすものはなかった。

URL、応答、スコープの判定、クォータを空けない本人の取り消し、`ConsentRevoked` の発行は変えていない。
移設の前に、本人に同意のないクライアントの撤回が HTTP 越しに 404 と `consent_not_found` で拒否されるテストを足した。
`check-status-drift` は、ヘルパー関数が書く応答を `unverified` として扱うため、今回の 404 の欠落を検出していなかった。
