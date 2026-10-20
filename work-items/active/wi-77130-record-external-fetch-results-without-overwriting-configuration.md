---
status: in_progress
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-21
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/workloadidentity/features/trust-configuration.md, requirement: REQ-WORKLOADIDENTITY-008, impact: conforms }
  - { path: docs/modules/authentication/features/federation.md, requirement: REQ-AUTHENTICATION-025, impact: modifies }
  - { path: docs/modules/authentication/features/federation.md, requirement: REQ-AUTHENTICATION-041, impact: modifies }
  - { path: spec/modules/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.RefreshIdentityProviderMetadata }
  - { path: spec/modules/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.PreviewIdentityProviderMetadata }
  - { path: spec/modules/authentication/models.tsp, symbol: IdMagic.Contract.IdentityProviderMetadataPreview }
  - { path: spec/modules/authentication/models.tsp, symbol: IdMagic.Contract.IdentityProviderMetadataPreviewRequest }
  - { path: spec/modules/authentication/models.tsp, symbol: IdMagic.Contract.IdentityProviderMetadataInvalidError }
primary_use_cases:
  - id: jwks-refresh-keeps-concurrent-edits
    requirement: REQ-WORKLOADIDENTITY-008
    observable_result: JWKS の取得中に無効化と設定の編集を行うと、取得時刻だけが記録され、無効化と設定の編集が残る。
    boundary: unit
    test: { path: backend/workloadidentity/internal/usecases/admin_trust_bundles_test.go, name: TestJWKSRefreshPreservesConcurrentConfigurationChanges, task: test-go-race }
    fault_model: 取得前の信頼設定を丸ごと保存し、その間の無効化と編集を書き戻す。
  - id: discovery-preview-does-not-save
    requirement: REQ-AUTHENTICATION-041
    observable_result: 管理者が issuer からメタデータの候補を取得しても接続は保存されず、取得中の無効化と編集が残る。
    boundary: adapter
    test: { path: backend/authentication/internal/federation/handlers_http/metadata_preview_test.go, name: TestMetadataPreviewPreservesConcurrentConfigurationChanges, task: test-go-race }
    fault_model: 入力支援の取得処理が、取得前の接続を保存してしまう。
  - id: discovery-preview-needs-explicit-save
    requirement: REQ-AUTHENTICATION-041
    observable_result: 編集フォームで候補を取得しても入力と保存済みの設定は変わらず、適用ボタンで候補を入力へ反映し、保存ボタンで初めて接続が更新される。
    boundary: adapter
    test: { path: frontend/src/features/admin-identity-providers/AdminIdentityProviderFormPage.test.tsx, name: '取得した候補は適用と保存を明示するまで接続を書き換えない', task: test-ui-unit }
    fault_model: 候補を取得しただけで設定を保存するか、管理者の確認を経ずにフォームを上書きする。
---

# 外部取得の結果を、設定を上書きせずに記録する

## 動機

次の二つの管理操作は、設定を読んでから外部のメタデータを取得し、その後に設定を丸ごと保存する。

| 操作 | 実装 | 本当に変える値 |
| --- | --- | --- |
| トラストバンドルの JWKS の確認 | `backend/workloadidentity/internal/usecases/admin_trust_bundles.go:RefreshWorkloadTrustBundleJWKS` | `jwks_cached_at` |
| IdP 接続のメタデータの更新 | `backend/authentication/internal/federation/handlers_http/routes.go:refresh` | Discovery で得たエンドポイントなどのメタデータ |

外部の取得には数秒かかり得る。
その間に別の管理者が同じ設定を無効化すると、取得後の保存が読んだ時点の状態を書き戻し、無効にしたトラストバンドルや IdP 接続が有効に戻る（REQ-WORKLOADIDENTITY-008）。
同じ窓で、名前、シークレットなどの設定の変更も失われる。

管理者どうしの同じ項目の編集の競合は [wi-605](wi-605-detect-lost-updates-with-etag-and-if-match.md) が版で扱う。
この二つは、外部取得を行う処理が設定全体を書き戻す問題である。
JWKS の確認は書く列を絞り、IdP の Discovery は設定の入力支援として保存を取り除く。

## 対象範囲

- JWKS の確認は、取得時刻だけを書く専用の永続化操作へ置き換える。メモリと PostgreSQL の実装と共有の契約テストを揃える。
- IdP の Discovery は、取得元の issuer から検証済みの候補を返す入力支援へ変え、接続の保存を取り除く。画面で候補を確認して適用し、通常の作成または編集で保存する。
- 外部取得の間に無効化した設定が、取得後も無効のままであることを検査する。
- 修正後に、[同時の更新と排他](../../docs/design/data/database.md#同時の更新と排他)の適用状況から二つの操作を除く。

## 対象外

- 管理者どうしの同じ項目の編集の競合（wi-605）。
- Authentication が User を丸ごと保存する問題（[wi-98381](../done/wi-98381-update-authentication-state-without-overwriting-the-user.md)）。
- 所有者の停止に伴う Agent の無効化（[wi-97231](wi-97231-disable-owned-agents-with-a-conditional-update.md)）。

## 設計

JWKS の記録はログインセッションの `RevokeAuthenticationSession` と `TouchAuthenticationSession` と同じ形にする。
IdP の Discovery は設定の書き手から外す。

- トラストバンドル：`RecordJWKSFetched(ctx, tenantID, id, at)` が `jwks_cached_at` だけを書く。
- IdP 接続：Discovery の結果を保存せず返す。管理者がフォームで候補を適用し、通常の編集で保存する。

規則は[同時の更新と排他](../../docs/design/data/database.md#同時の更新と排他)が定める。

### 着手時の調査と操作

JWKS の確認は `jwks_cached_at` だけを変え、`updated_at` を変えない。
`RecordJWKSFetched(ctx context.Context, tenantID, id string, at time.Time) error` は、テナントと ID が一致する既存の行の取得時刻だけを書く。
削除済みの行を作り直さず、メモリも同じ排他の中で該当する値だけを変える。
取得と記録は既存と同じ順で行い、記録の失敗時は成功イベントを発行しない。

IdP の `RefreshDiscovery` は `authorization_endpoint`、`token_endpoint`、`jwks_uri`、`metadata_refreshed_at`、`updated_at` を変える。
三つのエンドポイントは管理者も編集するため、列の限定だけでは同じ列の編集を保護できない。
取得元を比較して 409 を返す案は、設定と取得結果の所有が曖昧な構造を残すという利用者の指摘を受けて取り下げた。
名前、状態、シークレット、マッピングなどは Discovery の記録の対象に含めない。
既存のモジュール境界と組み立ては保ち、新しい依存は追加しない。

### IdP の再設計

利用者と比較した結果、Discovery を入力支援にする案を採用する。
設定の保存は管理者の操作だけが行い、取得処理には保存先を渡さない。
永続的な取得方式とメタデータの保存先を分ける案は、設定の同期と読み取り時の優先規則が必要になるため採らない。
ブラウザーが上流へ直接取得する案は相手の CORS に依存し、既存の取得先と信頼の検証も使えなくなるため採らない。

`Client.Discover(ctx context.Context, issuer string) (*MetadataPreview, error)` は、issuer、認可とトークンのエンドポイント、JWKS URI を検証して返し、接続を書き換えない。
`POST /api/admin/v1/identity-providers/metadata-preview` は issuer を本文で受け、既存と同じ対話セッションの管理者認証と CSRF 検査を適用する。
接続 ID を受けないため、未作成の接続にも使える。
旧 `/{provider_id}/refresh` は初回公開前のため削除し、成功が自動保存を意味する旧操作は残さない。
画面は候補と取得元を表示し、適用操作でエンドポイントだけをフォームへ反映する。
取得後にフォームの issuer またはプロトコルが変わったら候補を無効にし、古い取得結果を適用しない。
フォームの保存は既存の作成と更新を使い、信頼の根拠が変わる編集で Disabled にする既存の規則を保つ。
既存の保存された接続と DB の列は移行せず、取得方式やキャッシュの永続状態は追加しない。

### 規範の差分

REQ-AUTHENTICATION-025 から、接続 ID を指定したメタデータの再取得と、その取得時の拒否を取り除く。
REQ-AUTHENTICATION-041 を追加し、次を宣言する。

- 対話セッションの管理者が HTTPS の issuer から OIDC のメタデータの候補を要求したとき、Authentication は、Discovery と JWKS の検証を通った issuer と三つのエンドポイントを 200 で返し、接続を保存しない。
- issuer の不一致、取得先の拒否、上流の取得または鍵の検証の失敗は 400 と `metadata_invalid` で拒否し、接続を変えない。
- JSON が不正か issuer が空なら 400 と `invalid_request` で拒否する。
- API アクセストークンからの要求は既存と同じ 403 と `insufficient_scope` で拒否する。
- 管理画面は取得した候補を表示し、管理者が適用してから保存するまで設定を更新しない。候補の取得後に issuer またはプロトコルが変わったら、その候補を適用させない。

取得時の自動保存を取り除く仕様変更は、利用者に確認済みである。
ログインが保存済みの設定だけを使うことと、検査に使う HTTPS、issuer の完全一致、JWKS の検証は保つ。

## 計画

1. 外部取得を待つ間に無効化を挟むテストで、二つの操作の故障を RED にする。
2. 専用の永続化操作を実装し、GREEN にする。
3. データ設計の適用状況を更新する。

## タスク

- [x] T001 [Acceptance] JWKS の取得ポートへ無効化と編集を挟む故障を RED にし、IdP の候補取得とフォームの取得、適用、保存を RED にする。
- [x] T002 [App] JWKS の時刻だけを書く操作と、保存しない Discovery の候補取得を実装する。
- [x] T003 [Docs] 機能仕様と TypeSpec、データ設計の適用状況を同期する。
- [x] T004 [Verify] 変更行の変異と手動の故障注入、共有契約、逆依存、HTTP 契約、UI と正式なブラウザーの経路を検査する。

## 検証

- メモリと PostgreSQL の共有の契約テストで、専用の操作がほかの列を変えないことを確かめる。PostgreSQL の検査はサンドボックスの外で実行する。
- `mise run test-go-changed`、`mise run lint-go`、`mise run check-links`。
- JWKS の受け入れは既存の取得ポートに無効化と編集を挟む `TestJWKSRefreshPreservesConcurrentConfigurationChanges` を `mise run test-go-test` で実行する。ルートの配線を変えないため、同じ故障の E2E は追加しない。
- `mise run report-contract-evidence -- --changed --base-revision main --target backend/workloadidentity/internal/usecases/admin_trust_bundles.go --target backend/authentication/internal/federation/handlers_http/routes.go` で予定する HTTP の影響と契約参照を照合する。
- メモリと PostgreSQL の共有契約で、取得結果以外の列、テナント境界、存在しない行を検査する。
- 変更行の変異は `mise run check-go-mutation-tool` の後に `mise run test-go-mutation` の第 5 引数へ着手前の `2c4c810cd` を渡して調べる。取得前の全体保存に戻す故障はツールで表現できないため手で注入する。
- 最終検査は `test-go-changed` で両モジュールと逆依存を、`lint-go`、`check-boundaries` で Go と依存を、`check-contract-drift`、`check-status-drift`、`check-published-api-compat` で HTTP 契約を、`check-work-items`、`check-links`、`check-spec-impact` で宣言と参照を確認する。
- UI は `test-ui-unit-file` で候補の取得、適用、保存と古い候補の排除を、`verify-ui` で型とビルドを、`test-ui-e2e-file` で正式な作成と編集フォームを確認する。仕様と経路の変更後に `render-docs` と `generate-contract` を実行する。
- ブラウザーは `test-ui-e2e-file -- tests/e2e/identity-provider-metadata.spec.ts` で実サーバーの取得先の拒否と、手動設定の明示保存を観測する。候補の成功と適用は HTTP 契約と UI の単体検査に分け、上流の公開ネットワークには依存しない。
- ルーティングの変更後に `generate-route-reference` を実行し、`check` で経路の参照も照合する。

## リスク

候補の取得中に issuer を変えると、古い応答が新しいフォームへ入りうる。
画面は issuer とプロトコルの変更で候補の状態を破棄する。
管理者どうしの通常の保存の競合は wi-605 に残す。

## 要約

トラストバンドルの JWKS の確認は、`RecordJWKSFetched` で取得時刻だけを記録する形に変えた。
取得中の無効化と設定の編集を保持し、メモリと PostgreSQL はテナントと ID が一致する既存の行だけを更新する。
削除済みの行は作り直さない。

IdP の Discovery の結果と管理者の設定は同じエンドポイントの列を共有していたため、列を限定するだけでは書き手の競合が残った。
利用者との検討に基づき、取得元の変更で 409 を返す案と、取得方式とメタデータを永続化する案は採らず、Discovery を設定の入力支援にした。
初回公開前の `RefreshIdentityProviderMetadata` を削除し、issuer を受けて保存せず候補だけを返す `PreviewIdentityProviderMetadata` へ置き換えた。
REQ-AUTHENTICATION-025 から自動保存する再取得を外し、REQ-AUTHENTICATION-041 に候補の取得、拒否、明示的な適用と保存を宣言した。

サーバーは既存の HTTPS、取得先、issuer の完全一致と JWKS の検証を保ち、取得処理に接続や保存先を渡さない。
作成と編集のフォームは、取得した候補を表示し、管理者が適用するまで入力を変えず、通常の保存操作だけが接続を更新する。
issuer またはプロトコルの変更は候補を破棄し、古い応答を新しい設定へ適用させない。
接続の保存済みデータと DB の列は移行せず、取得方式やキャッシュの永続状態も増やしていない。

変更行への構文上の変異と、全体保存への差し戻し、SQL による状態の上書き、古い候補の保持という手動の故障を区別した。
文字列の連結を減算へ変える変異はコンパイル不能であり、検出能力の証拠に含めない。
PostgreSQL の変更行では構文上の変異を生成できないため、SQL が更新する列の故障を共有契約で観測する。
管理者どうしの通常の保存の競合は wi-605、所有者の停止に伴う Agent の無効化は wi-97231 に残す。
