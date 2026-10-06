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
  level: release_note
  reason: デバイス認可の `device_code` の交換が、`offline_access` を含まないスコープではリフレッシュトークンを返さなくなる。期限を過ぎた `user_code` の拒否が `expired_token` で拒否されるようになる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-87724-fix-the-device-flow-refresh-token-and-expired-denial.md }
initial_context:
  specification:
    - docs/domain/oauth2/token/README.md#REQ-OAUTH2-021
    - docs/domain/oauth2/device/README.md#REQ-OAUTH2-027
    - docs/domain/oauth2/token/acceptance.feature.md
    - docs/domain/oauth2/device/acceptance.feature.md
  typespec: []
  source:
    - backend/oauth2/device/usecases/device_flow.go
    - backend/oauth2/token/usecases/exchange_code.go
    - backend/oauth2/handlers_http/token_handler.go
  tests:
    - backend/oauth2/device/usecases/device_flow_test.go
    - backend/oauth2/device/usecases/device_flow_resource_indicator_test.go
    - backend/oauth2/device/usecases/common_test.go
    - backend/oauth2/handlers_http/device_code_resource_indicator_test.go
  stop_before_reading: [backend/oauth2/db_postgres, spec, frontend]
affected_spec:
  - { path: docs/domain/oauth2/token/README.md, requirement: REQ-OAUTH2-021 }
  - { path: docs/domain/oauth2/device/README.md, requirement: REQ-OAUTH2-027 }
primary_use_cases:
  - id: device-exchange-without-offline-access
    requirement: REQ-OAUTH2-021
    observable_result: スコープに `offline_access` を含まない `device_code` の交換は、アクセストークンと ID トークンを返し、リフレッシュトークンを返さず、保存せず、`RefreshTokenIssued` を発行しない。`offline_access` を含む交換は、リフレッシュトークンを返して保存し、`RefreshTokenIssued` を発行する。
    boundary: acceptance
    test: { path: backend/oauth2/device/usecases/device_flow_test.go, name: TestExchangeDeviceCodeIssuesARefreshTokenOnlyForOfflineAccess, task: test-go-race }
    fault_model: デバイスの交換がスコープを見ずに、常にリフレッシュトークンを生成して保存する。
  - id: device-deny-expired-user-code
    requirement: REQ-OAUTH2-027
    observable_result: 有効期間を過ぎた `user_code` の拒否は `expired_token` で拒否され、記録は `Denied` にならず、`DeviceAuthorizationDenied` を発行しない。
    boundary: acceptance
    test: { path: backend/oauth2/device/usecases/device_flow_test.go, name: TestDenyUserCodeRefusesAnExpiredUserCode, task: test-go-race }
    fault_model: 拒否の経路が有効期間を確かめずに、記録を `Denied` にする。
---

# デバイス認可で、offline_access のない交換にリフレッシュトークンを発行せず、期限切れの拒否を受け付けない

## 動機

デバイス認可の `device_code` の交換は、スコープに `offline_access` を含まなくてもリフレッシュトークンを発行する。
REQ-OAUTH2-021 は、リフレッシュトークンを `offline_access` を付与したときだけ発行すると定め、認可コードの交換はそれに従っている。
また、期限を過ぎた `user_code` の拒否は期限を確かめずに記録を `Denied` にするが、DeviceCodeFlow の状態遷移表は `拒否：400 expired_token` を求める。

## 対象範囲

- `device_code` の交換で、`offline_access` を含むスコープのときだけリフレッシュトークンを発行する。
- 期限を過ぎた `user_code` の拒否を、承認と同じく 400 と `expired_token` で拒否する。

## 対象外

- デバイス認可のほかの振る舞い。
- `offline_access` を要求したクライアントが `refresh_token` のグラントを許可していない場合の扱い。認可コードの交換も今はグラントを確かめておらず、両方の交換で同じ判断が要る。

## 設計

### 要件の差分

| 要件 | 変更 | 変更後の要件文 |
| --- | --- | --- |
| REQ-OAUTH2-021 | 交換の対象を `device_code` へ広げる | `offline_access` を含むスコープの認可コードか `device_code` を交換したとき、OAuth2 は、リフレッシュトークンを返し、`RefreshTokenIssued` を発行する。<br>`offline_access` を含まないスコープの認可コードか `device_code` を交換したとき、OAuth2 は、リフレッシュトークンを返さない。 |
| REQ-OAUTH2-027 | 交換の条項でリフレッシュトークンを `offline_access` のスコープに限る | クライアントが `Approved` の `device_code` を交換したとき、OAuth2 は、記録を `Exchanged` にし、アクセストークンと ID トークンを返し、`offline_access` のスコープにはリフレッシュトークンを返し、`AccessTokenIssued` を発行する。 |

期限を過ぎた `user_code` の拒否は、REQ-OAUTH2-027 の「有効期間を過ぎた `device_code` か `user_code` を受けた場合、OAuth2 は、400 と `expired_token` で拒否する」と状態遷移表の `Expired` の行がすでに定めている。
実装をこれに合わせる (c) であり、要件文は変えない。

### 実装

| 操作 | 変更 |
| --- | --- |
| `ExchangeDeviceCode(ctx, deps ExchangeDeviceCodeDeps, in ExchangeDeviceCodeInput, now time.Time) (*ExchangeDeviceCodeResult, error)` | 認可コードの交換（`exchange_code.go`）と同じく、`rec.Scopes` が `offline_access` を含むときだけ `GenerateInitialRefreshToken`、保存、`RefreshTokenIssued`、`IssuedFamilyID` の記録を行う。含まないとき `RefreshToken` は空文字列で、HTTP の層はすでに `refresh_token` を省く |
| `DenyUserCode(ctx, deps VerifyUserCodeDeps, userCode, sub string, now time.Time) error` | テナントの確認の後、`ApproveUserCode` と同じく `domain.IsDeviceExpired(rec, now)` で `expired_token` を返し、記録を更新せず、イベントを発行しない |

時刻は既存どおり引数の `now` で受け、新しい作用はない。

### 実装で決めたこと

| 場所 | 決定 | 理由 |
| --- | --- | --- |
| `GrantsRefreshToken(scopes []string) bool`（`oauth2/token/domain`） | リフレッシュトークンを発行できるかの判定を一つの関数に置き、認可コードと `device_code` の交換の両方から使う | 長期の資格情報を発行する安全性条件であり、二つの交換で食い違ったことがこの不具合の原因である。抽出は振る舞いを変えない構造変更として先のコミットにした |
| 既存のテスト | `TestDeviceFlowPollingAndReplay` の `scope=openid` の交換からリフレッシュトークンの表明を外し、resource indicator のテストは `offline_access` を要求して記録への `resource` の伝播を確かめる | どちらも、`offline_access` なしでリフレッシュトークンを返す不具合を前提に書かれていた |

### 仕様の漏れを探す観点で見つけたこと

| 観点 | 見つけたこと | 分類 |
| --- | --- | --- |
| 同じ種類の操作 | 承認と拒否で期限の判定が食い違う | (c) 拒否に承認と同じ判定を置く |
| 同じ種類の操作 | 認可コードとデバイスの交換で、リフレッシュトークンの発行条件が食い違う | (a) REQ-OAUTH2-021 を両方の交換の規則にする |
| 認可 | `offline_access` を要求したクライアントの `refresh_token` のグラントを、どちらの交換も確かめない | 対象外へ回す |

## 計画

1. 変更する箇所の現在の振る舞いを特性化テストで固定する。
2. 要件の差分を仕様へ反映する。
3. 二つの食い違いのテストを書き、RED を確かめる。
4. 実装を直し、特性化テストを要件のテストへ移す。

## タスク

- [x] T000 [Characterize] `TestCharacterizeExchangeDeviceCodeRefreshToken` と `TestCharacterizeDenyUserCode` で、交換のリフレッシュトークン（記録、イベント、`IssuedFamilyID`）と拒否の結果を固定する。`test-go-mutation` で、変更する箇所（リフレッシュトークンの発行と拒否の経路）への変異をすべて検出することを確かめた。
- [x] T001 [Acceptance] 二つの食い違いを RED で確認する。`TestExchangeDeviceCodeIssuesARefreshTokenOnlyForOfflineAccess/openid` はリフレッシュトークンと `IssuedFamilyID` が残って失敗し、`TestDenyUserCodeRefusesAnExpiredUserCode/after_the_lifetime` は `err = <nil>, want "expired_token"` で失敗した。
- [x] T002 [App] GREEN にする。リフレッシュトークンの発行条件をドメインの `GrantsRefreshToken` に集め、認可コードの交換も同じ関数を使う。
- [x] T003 [Verify] 変更を検証する。

## 検証

- 各 RED と GREEN：`mise run test-go-test -- ./backend/oauth2/device/usecases <test>`
- 振る舞いが GREEN になった後：`mise run test-go-package -- ./backend/oauth2/device/usecases`、`mise run test-go-changed`、`mise run lint-go`
- 変更への耐性：`mise run test-go-mutation -- backend/oauth2/device/usecases`
- `mise run verify`

## リスク

- 既存のデバイス認可のクライアントがリフレッシュトークンに依存している場合、`offline_access` を要求するよう変える必要がある。リリースノートで知らせる。

## 完了

- **Completed At**: 2026-10-06
- **Summary**:
  `mise run spec-diff -- main` の結果、REQ-OAUTH2-021 と REQ-OAUTH2-027 の要件文が変わった。
  REQ-OAUTH2-021 は、リフレッシュトークンを `offline_access` のスコープのときだけ発行する規則を、認可コードと `device_code` の両方の交換に定める。REQ-OAUTH2-027 の交換の条項は、`offline_access` のスコープにだけリフレッシュトークンを返すと定める。
  `device_code` の交換は、スコープに `offline_access` を含まないとき、アクセストークンと ID トークンだけを返し、リフレッシュトークンを保存せず、`RefreshTokenIssued` を発行しない。発行の判定はドメインの `GrantsRefreshToken` に置き、認可コードの交換と共有する。
  有効期間を過ぎた `user_code` の拒否は、承認と同じく `expired_token` で拒否され、記録は `Denied` にならず、`DeviceAuthorizationDenied` を発行しない。
- **Primary Use Case Evidence**:
  - id: device-exchange-without-offline-access
    red: 実装前に TestExchangeDeviceCodeIssuesARefreshTokenOnlyForOfflineAccess/openid が、リフレッシュトークンと `IssuedFamilyID` が残ることで失敗した。
    fault_injection: 交換の関数 `ExchangeDeviceCode` の `GrantsRefreshToken` の条件を外す（変更前の実装）と、同じテストが同じ失敗をした。
  - id: device-deny-expired-user-code
    red: 実装前に TestDenyUserCodeRefusesAnExpiredUserCode/after_the_lifetime が `err = <nil>, want "expired_token"` で失敗した。
    fault_injection: 拒否の関数 `DenyUserCode` の `IsDeviceExpired` の判定を外す（変更前の実装）と、同じテストが同じ失敗をした。
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/oauth2/device/usecases` の結果は 45 件検出・4 件生存・1 件未被覆で、変更の前後で同じだった。
  生き残りは、承認の保存の失敗の分岐、ポーリング間隔の境界の比較、送信者制約の二つの分岐にあり、この作業で変えていない。変更した拒否の経路とリフレッシュトークンの発行の分岐には生き残りがない。
  条件の除去は変異器が表せないので、上の `fault_injection` の 2 件を手で確かめた。
  特性化テスト `TestCharacterizeExchangeDeviceCodeRefreshToken` と `TestCharacterizeDenyUserCode` は、変更前に変更する箇所への変異をすべて検出した。変更後の分類は、`offline_access` のときの発行（記録、イベント、`IssuedFamilyID`）と有効期間内の拒否を (a) とし、`//spec:covers` を付けた二つの要件のテストへ移して特性化テストを消した。`offline_access` なしの発行と期限切れの拒否の受理は (c) として直した。
  手で解析する入力はなく、ファジングの対象は足していない。
- **Verification Results**:
  - `mise run test-go-changed` - 成功
  - `mise run lint-go` - 成功
  - `mise run spec-diff -- main` - REQ-OAUTH2-021 と REQ-OAUTH2-027 の変更
  - `mise run verify` - 成功
  - `mise run test-ui-e2e` - 成功（39 件）。初回は、デバイス認可と関係のない管理画面のロゴのアップロードが WebView の `evaluate() is already pending` で 1 件失敗し、再実行で通った
