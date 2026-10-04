---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p1
depends_on: [wi-35451-rewrite-remaining-context-specifications-in-the-requirement-format]
change_kind: bugfix
evidence_policy: risk-based-v4
documentation_impact:
  level: removal_notice
  reason: 発行されないイベントの型 4 件を契約と監査の検索の絞り込みから外し、第二要素の失敗の記録を AuthenticationFailed から MfaChallengeFailed へ移すため、その名前で監査を検索していた運用者には移行が必要になる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-37490-emit-or-retire-declared-events-that-nothing-emits.md }
    - { kind: upgrade_note, path: docs/releases/upgrades/wi-37490-emit-or-retire-declared-events-that-nothing-emits.md }
initial_context:
  specification:
    - docs/domain/authentication/mfa/README.md#REQ-AUTHENTICATION-015
    - docs/domain/authentication/totp/README.md#REQ-AUTHENTICATION-017
    - docs/domain/authentication/webauthn/README.md
    - docs/domain/authentication/security-notification/README.md#REQ-AUTHENTICATION-031
    - docs/domain/authentication/design/authentication-events.md
    - docs/domain/provisioning/connection/README.md#REQ-PROVISIONING-002
    - docs/domain/oauth2/authorization/README.md#REQ-OAUTH2-053
  typespec:
    - IdMagic.Contract.MfaChallengeIssued
    - IdMagic.Contract.ProvisioningConnectionUpdated
    - IdMagic.Contract.AuthorizationDetailsRejected
  source:
    - backend/authentication/domain/events.go
    - backend/authentication/usecases/retention.go
    - backend/oauth2/handlers_http/authorize_login.go
    - backend/oauth2/handlers_http/authorize_second_factor.go
    - backend/oauth2/handlers_http/authorize_handler.go
    - backend/oauth2/authorization/usecases/authorize.go
    - backend/oauth2/authorization/usecases/push_authorization_request.go
    - backend/oauth2/usecases/authorization_details.go
    - backend/provisioning/usecases/admin.go
    - backend/provisioning/domain/events.go
    - tools/check/src/unspecified-vocabulary.ts
  tests:
    - backend/provisioning/usecases/admin_events_test.go
    - backend/shared/http/server_http/routes_e2e_test.go
  stop_before_reading: [frontend/src/features/account]
affected_spec:
  - { path: docs/domain/authentication/mfa/README.md, requirement: REQ-AUTHENTICATION-015 }
  - { path: docs/domain/authentication/totp/README.md, requirement: REQ-AUTHENTICATION-017 }
  - { path: docs/domain/authentication/webauthn/README.md, requirement: REQ-AUTHENTICATION-040 }
  - { path: spec/contexts/authentication/models.tsp, symbol: IdMagic.Contract.MfaChallengeIssued }
  - { path: docs/domain/provisioning/connection/README.md, requirement: REQ-PROVISIONING-002 }
  - { path: docs/domain/oauth2/authorization/README.md, requirement: REQ-OAUTH2-053 }
primary_use_cases:
  - id: mfa-challenge-issued
    requirement: REQ-AUTHENTICATION-015
    observable_result: パスワードが正しく第二要素を求められたログインで、提示した要素の種類を載せた MfaChallengeIssued が記録される。
    boundary: acceptance
    test: { path: backend/shared/http/server_http/routes_e2e_test.go, name: TestBrowserAuthorizationFlowRequiresTOTPWhenPolicyRequiresMFA, task: test-go-race }
    fault_model: セッションを第二要素の待ちにする経路の一つが、MfaChallengeIssued を発行しない。
  - id: totp-second-factor-outcome
    requirement: REQ-AUTHENTICATION-017
    observable_result: 誤った TOTP のコードは AuthenticationFailed ではなく MfaChallengeFailed を、正しいコードは MfaChallengeSucceeded と UserAuthenticated を記録する。
    boundary: acceptance
    test: { path: backend/shared/http/server_http/routes_e2e_test.go, name: TestTOTPSecondFactorRecordsChallengeOutcome, task: test-go-race }
    fault_model: 第二要素の失敗が AuthenticationFailed のまま残るか、成功が MfaChallengeSucceeded を記録しない。
  - id: webauthn-second-factor-failure
    requirement: REQ-AUTHENTICATION-040
    observable_result: 検証できない WebAuthn のアサーションは 401 と invalid_webauthn で拒否され、factorType=webauthn の MfaChallengeFailed を記録する。
    boundary: acceptance
    test: { path: backend/shared/http/server_http/routes_e2e_test.go, name: TestWebAuthnSecondFactorFailureRecordsChallengeFailed, task: test-go-race }
    fault_model: WebAuthn の失敗が AuthenticationFailed のまま残るか、要素の種類を取り違える。
  - id: provisioning-connection-change-events
    requirement: REQ-PROVISIONING-002
    observable_result: 接続の設定の更新、無効化、削除が、保存の後にそれぞれ ProvisioningConnectionUpdated、ProvisioningConnectionDisabled、ProvisioningConnectionDeleted を記録する。
    boundary: unit
    test: { path: backend/provisioning/usecases/admin_events_test.go, name: TestConnectionChangesEmitTheirEventAfterSaving, task: test-go-race }
    fault_model: 資格情報だけの更新で Updated を出す、無効化を Updated と区別しない、保存に失敗した削除で Deleted を出す。
  - id: authorization-details-rejected
    requirement: REQ-OAUTH2-053
    observable_result: 登録済みのクライアントの認可か PAR の要求の authorization_details を拒否すると、reason=invalid_authorization_details の AuthorizationDetailsRejected を記録する。
    boundary: unit
    test: { path: backend/oauth2/authorization/usecases/authorization_details_rejected_test.go, name: TestAuthorizationDetailsRefusalEmitsRejected, task: test-go-race }
    fault_model: 拒否が発行を伴わないか、未登録のクライアントからの要求でも発行する。
---

# 宣言しているのに発行されないドメインイベントを、発行するか宣言から外す

## 動機

wi-35451 で語彙を検査したところ、宣言しているのにどの経路も発行しないイベントが見つかった。

| Context | 発行されないイベント |
| --- | --- |
| Authentication | `AuthenticationStepCompleted`、`AuthenticationStepFailed`、`MfaChallengeIssued`、`MfaChallengeSucceeded`、`MfaChallengeFailed`、`SessionStarted`、`SessionRefreshed`、`SessionImpersonationStarted`、`SessionImpersonationEnded` |
| Provisioning | `ProvisioningConnectionUpdated`、`ProvisioningConnectionDisabled`、`ProvisioningConnectionDeleted` |
| OAuth2 | `AuthorizationDetailsRejected` |

## 対象範囲

- イベントごとに、要件に書いて発行するか、宣言、保持期間、監査の検索の絞り込みから外すかを決めて実装する。
- 語彙の検査の許容リストから、扱ったイベントを外す。

## 対象外

- 新しいイベントの追加。
- `SessionImpersonationStarted` と `SessionImpersonationEnded`。管理者が User になりすます機能そのものがないので、発行する経路がない。機能の新設は wi-59470 が扱い、二つのイベントと通知の種別 `impersonation` の宣言はそれまで残す。
- Token Exchange と CIBA の承認での `authorization_details` の拒否。REQ-OAUTH2-053 は認可と PAR の要求を扱い、その二つの経路の拒否は別の要件の範囲である。
- 復旧コードによる第二要素。`MfaFactorType` に含まれない要素であり、成功は `BackupCodeConsumed` と `UserAuthenticated` が、失敗は `AuthenticationFailed` が引き続き記録する。

## 設計

### イベントごとの扱い

| イベント | 扱い | 判断の根拠 |
| --- | --- | --- |
| `MfaChallengeIssued` | 発行する | パスワードが正しく第二要素の待ちに入ったという事実を、ほかのどのイベントも記録しない。パスワードの漏えいの兆候や、MFA の要求の連打を検知する手掛かりになる |
| `MfaChallengeSucceeded` | 発行する | 第二要素の種類を構造化された項目で残す。完了の `UserAuthenticated` と並べて発行する |
| `MfaChallengeFailed` | 発行する | 第二要素の失敗は `AuthenticationFailed` から置き換える。`AuthenticationFailed` はユーザー名を確定できない第一要素の失敗の記録であり、第二要素の失敗では `username` の項目に `user_id` を入れていた |
| `AuthenticationStepCompleted`、`AuthenticationStepFailed` | 外す | パスワードの段の成功は `MfaChallengeIssued` か `UserAuthenticated` が、失敗は `AuthenticationFailed` が記録する |
| `SessionStarted` | 外す | 項目（`sessionId`、`amr`、`acr`、IP、User-Agent）が `UserAuthenticated` と同じで、同じ時点の同じ事実を二重に記録する |
| `SessionRefreshed` | 外す | セッションを更新する操作がない。`last_seen_at` の粗い更新は監査の対象にしない |
| `ProvisioningConnectionUpdated`、`ProvisioningConnectionDisabled`、`ProvisioningConnectionDeleted` | 発行する | 接続の変更は監査の利用者が追う対象であり、登録と資格情報のローテーションだけが記録されていた |
| `AuthorizationDetailsRejected` | 発行する | 拒否は `AuthorizationDetailsRequested` と対になる監査の記録である |

### 要件の差分

- REQ-AUTHENTICATION-015 に追加する：「実効のサインインポリシーが MFA を求めてログインセッションを第二要素の待ちにしたとき、Authentication は、User が使える第二要素の種類を載せた `MfaChallengeIssued` を発行する。」
- REQ-AUTHENTICATION-017 を変更する：正しいコードでは `MfaChallengeSucceeded`（`factorType=totp`）と `UserAuthenticated` を、誤ったコードでは `AuthenticationFailed` の代わりに `MfaChallengeFailed`（`factorType=totp`）を発行する。
- REQ-AUTHENTICATION-040 を新設する（WebAuthn の文書、利用者によるログインの第二要素）：検証に成功したアサーションで認証を成立させ、`MfaChallengeSucceeded`（`factorType=webauthn`）と `UserAuthenticated` を発行する。検証できないアサーションは 401 と `invalid_webauthn` で拒否し、`MfaChallengeFailed` を発行する。
- REQ-PROVISIONING-002 に追加する：資格情報と `status` 以外の設定を指定した更新か、`disabled` から `active` へ戻す更新では `ProvisioningConnectionUpdated` を、`status` を `disabled` にする更新では `ProvisioningConnectionDisabled` を、削除では `ProvisioningConnectionDeleted` を、それぞれ保存の後に発行する。一つの更新が複数に当たるときは、当たるものをすべて発行する。
- REQ-OAUTH2-053 に追加する：登録済みのクライアントの認可か PAR の要求の `authorization_details` を拒否したとき、理由 `invalid_authorization_details` の `AuthorizationDetailsRejected` を発行する。

### 契約の変更

- `MfaChallengeIssued.factorType` を `factorTypes: MfaFactorType[]` に変える。第二要素の待ちに入る時点では利用者はまだ要素を選んでいないので、提示した種類の集合が観測できる事実である。発行された実績がないので、移行は要らない。
- 外す 4 件は、TypeSpec のモデル、Go の型、保持期間の分類、監査の検索の候補（バックエンドと画面）から消す。

### 実装

- 第二要素の待ちに入る四つの経路（ログインのアプリケーション別ポリシー、既定のサインインポリシー、`/authorize` のステップアップ、認可の完了時のステップアップ）は、どれも `SessionManager.RequireFactor` を呼んだ直後に `MfaChallengeIssued` を発行する。四つの重複は、ハンドラー層の補助関数 `requireSecondFactor` にまとめる。
- 第二要素の成功は `finishSecondFactor` で発行する。`amr` の値（`otp`、`webauthn`）を `MfaFactorType` へ写し、写せない `rc` では発行しない。
- 仕様にない振る舞いの分類 (c)：`/authorize` のハンドラーは、クライアントを確かめる前に `authorization_details` を解析して拒否していた。未登録のクライアントの名で拒否のイベントを書けないよう、解析を `Authorize` のクライアント確認の後へ移す。`AuthorizeRequestInput.AuthorizationDetails` は生の文字列 `AuthorizationDetailsRaw` に変え、`AuthorizationDetailsRequested` は保存した要求の `AuthorizationDetails` から作る。これにより、不正な JSON の `authorization_details` と未登録のクライアントが重なった要求の応答は、`invalid_authorization_details` から `invalid_client` に変わる。
- 拒否の判定は `invalid_authorization_details` の `OAuthError` に限る。種類の保管先の障害は拒否ではないので発行しない。

## タスク

- [x] T001 [Spec] 要件の差分を各文書へ反映し、TypeSpec から 4 件を外し、`MfaChallengeIssued` の項目を変える。`mise run check-spec`、`mise run check-api-compat`。
- [x] T002 [App] Provisioning の更新、無効化、削除で発行する。RED と GREEN は `mise run test-go-test -- ./backend/provisioning/usecases TestConnectionChangesEmitTheirEventAfterSaving`。
- [x] T003 [App] 認可と PAR の `authorization_details` の拒否で発行する。RED と GREEN は `mise run test-go-test -- ./backend/oauth2/authorization/usecases TestAuthorizationDetailsRefusalEmitsRejected`。
- [x] T004 [App] 第二要素の待ち、成功、失敗で `MfaChallenge*` を発行する。RED と GREEN は `mise run test-go-test -- ./backend/shared/http/server_http 'TestBrowserAuthorizationFlowRequiresTOTPWhenPolicyRequiresMFA|TestTOTPSecondFactorRecordsChallengeOutcome|TestWebAuthnSecondFactorFailureRecordsChallengeFailed'`。
- [x] T005 [App] 外す 4 件を Go、保持期間、監査の検索の候補、画面から消し、語彙の検査の許容リストを減らす。代替検査は `mise run check-unspecified-vocabulary`。
- [x] T006 [Verify] `mise run verify` と `mise run test-ui-e2e` を通す。

## 検証

- `mise run check-unspecified-vocabulary`
- `mise run verify`
- `mise run test-ui-e2e`

## リスク

- 外すイベントを監査の検索の画面やエクスポートが名前で参照している。外す前に参照を検索し、同じ変更で消す。
- 第二要素の失敗の記録を `AuthenticationFailed` から置き換えるので、それを数えていた集計やメトリクスがあれば数が変わる。ログインの結果のメトリクスは `recordLoginOutcome` が別に数えるので変わらない。

## 完了

- **Completed At**: 2026-10-04
- **Summary**:
  `mise run spec-diff` は、main に対して REQ-AUTHENTICATION-040 の追加、REQ-AUTHENTICATION-015、REQ-AUTHENTICATION-017、REQ-OAUTH2-053、REQ-PROVISIONING-002 の変更、TypeSpec の `AuthenticationStepCompleted`、`AuthenticationStepFailed`、`SessionStarted`、`SessionRefreshed` の削除、`MfaChallengeIssued` の変更を報告した。
  ログインセッションを第二要素の照合の待ちにする四つの経路は、提示できる要素の種類を `factorTypes` に載せた `MfaChallengeIssued` を発行する。TOTP と WebAuthn の第二要素の成功は `MfaChallengeSucceeded` を、失敗は `AuthenticationFailed` の代わりに `MfaChallengeFailed` を発行する。復旧コードは要素の種類ではないので、従来どおり `AuthenticationFailed` と `BackupCodeConsumed` が記録する。
  WebAuthn のログインの第二要素の要件 REQ-AUTHENTICATION-040 を新設し、`webauthn_not_enrolled` も要件に書いた。
  Provisioning の接続の設定の更新、無効化への遷移、削除は、保存の後に `ProvisioningConnectionUpdated`、`ProvisioningConnectionDisabled`、`ProvisioningConnectionDeleted` を発行する。`DeleteConnection` は時刻を引数に取る。
  `/authorize` と PAR は、登録済みのクライアントの `authorization_details` を拒否したとき `AuthorizationDetailsRejected` を発行する。`/authorize` はクライアントを確かめた後に `authorization_details` を解析するようになり、`AuthorizeRequestInput` は生の文字列 `AuthorizationDetailsRaw` を受け取る。
  同じ事実をほかのイベントが記録する 4 件を、Go の型、保持期間の分類、監査の検索の候補（バックエンドと画面）から外した。語彙の検査の許容リストには、なりすましの 2 件だけが残る。なりすまし機能の新設は wi-59470 として起票した。
- **Primary Use Case Evidence**:
  - id: mfa-challenge-issued
    red: 第二要素の待ちに入っても発行しないため、TestBrowserAuthorizationFlowRequiresTOTPWhenPolicyRequiresMFA が `MfaChallengeIssued = [], want one for user_alice offering [totp]` で失敗した。
    fault_injection: 補助関数 `requireSecondFactor` の発行を無効にすると、同じテストが同じ表明で失敗した。
  - id: totp-second-factor-outcome
    red: 誤ったコードが `AuthenticationFailed` を発行していたため、TestTOTPSecondFactorRecordsChallengeOutcome が `AuthenticationFailed = [...], want none for a second-factor failure` で失敗した。
    fault_injection: 後処理 `finishSecondFactor` の `MfaChallengeSucceeded` の発行を無効にすると、同じテストが `MfaChallengeSucceeded = [], want one totp success for user_alice` で失敗した。
  - id: webauthn-second-factor-failure
    red: 検証できないアサーションが `AuthenticationFailed` を発行していたため、TestWebAuthnSecondFactorFailureRecordsChallengeFailed が `AuthenticationFailed = [...], want none for a second-factor failure` で失敗した。
    fault_injection: WebAuthn の失敗の要素の種類を `totp` に取り違えると、同じテストが `want one webauthn failure for user_alice` で失敗した。
  - id: provisioning-connection-change-events
    red: 更新が何も発行しないため、TestConnectionChangesEmitTheirEventAfterSaving が `settings update events = [], want [ProvisioningConnectionUpdated]` で失敗した。
    fault_injection: 無効化の判定から遷移の条件を外すと、同じテストが `re-disabling events = [ProvisioningConnectionDisabled], want none` で失敗した。
  - id: authorization-details-rejected
    red: 拒否が何も発行しないため、TestAuthorizationDetailsRefusalEmitsRejected が認可と PAR の各ケースで `rejections = [], want one ... with reason invalid_authorization_details` で失敗した。
    fault_injection: エラーの種類を問わず発行すると、同じテストの `a registry failure is not a refusal` が `rejections = [...]; want the store error and none` で失敗した。
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/provisioning/usecases` は 237 件の変異を見つけ、208 件を試して 194 件を検出し、14 件が生き残った（29 件は被覆外）。生き残った変異はすべて本件で変更していない行（`UpdateConnection` の既存の項目の代入、`validateCredential`、`StartFullResync`、`RetryTask`、`credentialMetadata`、`CaptureLifecycleEvent`、`adoptExistingUser`、`pushGroupMembers`、`deletedVersion`）にある。
  `mise run test-go-mutation -- backend/oauth2/authorization/usecases` は 42 件を試して 38 件を検出し、3 件が生き残った。いずれも本件で変更していない行（`Authorize` の `prompt` の判定と有効期限の計算、`CompleteLogin` の有効期限の計算）にある。
  変異器が表せない配線の故障は手で注入した。第二要素の待ちと成功での発行の無効化、WebAuthn の要素の種類の取り違え、無効化の遷移の条件の除去、拒否以外のエラーでの発行を、それぞれ対応するテストが検出した（Primary Use Case Evidence を参照）。
  WebAuthn の成功は、本物のアサーションを作れないのでブラウザーの経路では確かめていない。`amr` から要素の種類への写像は TestSecondFactorKindsMapToMfaFactorTypes が固定し、写像の後の発行は TOTP と共通の `finishSecondFactor` を通る。
- **Verification Results**:
  - `mise run check-unspecified-vocabulary` - 成功。許容リストを減らす前は、要件に現れるようになった 12 件の語彙を報告して失敗した。
  - `mise run verify` - 成功（1 回目は UI の整形と `affected_spec` の TypeSpec の記載漏れで失敗し、直した後に成功した）
  - `mise run test-ui-e2e` - 成功（1 回目は、本件と関係しないテナントのブランディングのロゴのテストが WebView の `evaluate() is already pending` で失敗した。再実行で 39 件すべてが成功した）
