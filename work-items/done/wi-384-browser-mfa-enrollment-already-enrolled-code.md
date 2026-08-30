---
status: completed
authors: [tn]
risk: low
created_at: 2026-08-19
change_kind: bugfix
priority: p3
depends_on: []
evidence_policy: risk-based-v2
initial_context:
  specification: [docs/contexts/authentication/scenarios.md]
  typespec:
    - IdMagic.Authentication.Operations.StartBrowserMfaEnrollment
    - IdMagic.Authentication.Operations.ConfirmBrowserMfaEnrollment
  source:
    - backend/oauth2/handlers_http/authorize_enrollment.go
    - backend/authentication/deps_http/account_helpers.go
    - backend/authentication/mfa/handlers_http/admin_mfa_enrollment_handler.go
    - frontend/src/features/auth-flow/MfaEnrollmentPage.tsx
    - frontend/src/features/auth-flow/MfaEnrollmentPage.i18n.ts
  tests:
    - backend/oauth2/handlers_http
    - frontend/src/features/auth-flow/AuthFlowPages.test.tsx
  stop_before_reading: [backend/sourcing, backend/provisioning]
affected_spec:
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.StartBrowserMfaEnrollment }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.ConfirmBrowserMfaEnrollment }
---

# ブラウザー経路の MFA 登録が「既に登録済み」を「登録が許可されていない」に畳んでいるのを解く

## Motivation

`writeBrowserEnrollmentError` (`backend/oauth2/handlers_http/authorize_enrollment.go`) は `ErrMfaEnrollmentNotAllowed` と `ErrMfaAlreadyEnrolled` を同じ分岐で受け、どちらも 403 `mfa_enrollment_not_allowed` を返す。管理 API はこの 2 つを区別し、後者を 409 `mfa_already_enrolled` として返す (`backend/authentication/mfa/handlers_http/admin_mfa_enrollment_handler.go`)。

利用者から見ると意味がまるで違う。「登録が許可されていない」は管理者に問い合わせる状況で、「既に登録済み」は何もしなくてよい状況である。同じ code で返す限り、ログイン画面は 2 つを別の文面で案内できない。

`wi-382` は T009 でこの分岐を調べ、403 という status 自体は正しい (要求元セッションに対する認可判断である) と結論した。残っているのは code の畳み込みだけである。

## Scope

- ブラウザー経路で `ErrMfaAlreadyEnrolled` に固有の code を返す。status は管理 API と同じ 409 が妥当かを判断し、決めた側を契約に書く。
- `StartBrowserMfaEnrollment` / `ConfirmBrowserMfaEnrollment` の error union に対応するモデルを加える。
- ログイン画面の i18n 辞書に対応する文面を足す。

## Out of Scope

- `mfa_enrollment_not_allowed` そのものの status。403 のままとする (`wi-382` T009 の判断)。
- `mfa_enrollment_expired` (403) の宣言。ブラウザー経路が返しているのに `StartBrowserMfaEnrollmentError403Body` は `AccessDeniedError` と `MfaEnrollmentNotAllowedError` しか並べていない。同じ種類の食い違いだが、宣言と実装の総当たりは [[wi-386-declared-status-code-audit]] が持つ。
- `MfaEnrollmentPage` が API 由来の失敗でサーバーの英語の `detail` をそのまま出していること。`mfa_already_enrolled` は本 work item で訳したが、他の code は据え置いた。画面全体の方針の問題なので別途扱う。

## Design

**status は 409 にする。** 管理 API (`admin_mfa_enrollment_handler.go`) と account API (`deps_http/account_helpers.go`) は既に同じ条件を 409 `mfa_already_enrolled` として返している。「既に登録済み」は要求元セッションに対する認可判断ではなく現在の状態との衝突なので、403 ではなく 409 が正しい。`wi-382` T009 が 403 を正当とした対象は `mfa_enrollment_not_allowed` の側で、そちらは変えない。

`MfaAlreadyEnrolledError` を `ProblemDetails` として新設し、両 operation の union に 409 を足す。応答の追加は非破壊なので `check-api-compat` は反応しない。

**画面。** 既定文面 (`startFailed` / `completeFailed`) は元から用意されており、Risk Notes が求めた確認は満たされている。ただし API 由来の失敗ではサーバーの `detail` が出るため、そのままでは英語が表示される。`mfa_already_enrolled` だけを訳した文面へ写す純関数 `enrollmentErrorMessage` を置き、他の code は従来どおり `detail` を出す。未知の code も既定文面には落ちないので、code が増えても無言にはならない。

## Verification

- `mise run verify`
- 手動確認: MFA 登録済みの user がブラウザー経路で登録を開始すると、「登録が許可されていない」とは別の code が返る。

## Risk Notes

ログイン途中の画面が返す code を増やすので、未知の code に落ちたときの既定文面が用意されていることを先に確かめる。fail-closed の側 (登録させない) は変えない。

## Tasks

- [x] T001 [Spec] `MfaAlreadyEnrolledError` を新設し、`StartBrowserMfaEnrollment` と
  `ConfirmBrowserMfaEnrollment` に 409 を足した。
- [x] T002 [Acceptance] ブラウザー経路が「既に登録済み」を別の code で返すことの受け入れ検査を RED で置いた。
  `TestBrowserMfaEnrollmentSeparatesAlreadyEnrolledFromNotAllowed`。
- [x] T003 [Adapters] `writeBrowserEnrollmentError` の畳み込みを解き、`ErrMfaAlreadyEnrolled` を
  409 `mfa_already_enrolled` で返すようにした。
- [x] T004 [UI] 辞書に `alreadyEnrolled` を足し、`enrollmentErrorMessage` で code から文面を選ぶようにした。
- [x] T005 [Verify] `mise run verify`。

## Completion

- **Completed At**: 2026-08-30
- **Summary**:
  `mise run spec-diff` は `added TypeSpec declarations: spec/contexts/authentication/models.tsp:MfaAlreadyEnrolledError` を返す。ブラウザー経路の MFA 登録が「登録が許可されていない」と「既に登録済み」を同じ 403 `mfa_enrollment_not_allowed` へ畳んでいたのを解き、後者を 409 `mfa_already_enrolled` として返すようにした。管理 API と account API が同じ条件に既に使っている status と code に揃えたので、3 つの接点が同じ条件を同じ形で報告するようになっている。契約側は両 operation に 409 を足した。応答の追加なので `check-api-compat` は破壊的変更を報告しない。画面は `mfa_already_enrolled` に対して訳した文面を出す。fail-closed の側は変えていない。
- **Acceptance RED Evidence**:
  - **Test**: `TestBrowserMfaEnrollmentSeparatesAlreadyEnrolledFromNotAllowed` (`backend/oauth2/handlers_http/authorize_enrollment_test.go`)
  - **Requirement**: N/A: 対応する `REQ-` シナリオは無い。規範は TypeSpec の `StartBrowserMfaEnrollment` / `ConfirmBrowserMfaEnrollment` が宣言する応答の集合で、本 work item はそこに 409 を足して実装を合わせた。
  - **Observed Failure**: 開始の側が `status=403 body={"type":"urn:idmagic:error:mfa_enrollment_not_allowed",...}, want 409`、確認の側が `status=403 type="urn:idmagic:error:mfa_enrollment_not_allowed", want 409 mfa_already_enrolled`。
  - **Detection Reason**: HTTP の境界で、同じ画面が出しうる 2 つの条件を並べて見る。「既に登録済み」が 409 `mfa_already_enrolled` になることだけを見るテストは、両方を 409 に倒した実装にも通ってしまうので、「登録待ちでないセッション」が 403 `mfa_enrollment_not_allowed` のままであることを対で固定した。この 3 つめの部分試験は当初から通っており、失敗が畳み込みだけに由来することを分けている。登録開始の応答本体 (`secret`) が返っていないことも確かめ、拒否が何も通していないことを状態の側からも見ている。
- **Unit RED Evidence**:
  - **Test**: `auth-flow pages > shows the localized already-enrolled notice instead of the server detail` (`frontend/src/features/auth-flow/AuthFlowPages.test.tsx`)
  - **Requirement**: N/A: 上と同じ理由で、対応する `REQ-` シナリオを持たない。
  - **Observed Failure**: 変異 (code の分岐を外し常に `cause.message` を返す) に対して `1 fail` (`shows the localized already-enrolled notice instead of the server detail`)。
  - **Detection Reason**: 主張は辞書の値 (`mfaEnrollmentPageDictionary.en.alreadyEnrolled`) を参照しており、訳文を検査に直書きしていない。既定文面 (`startFailed`) が出ていないことも併せて見るので、code を読まずに既定へ落とすだけの実装は区別される。対になる 2 つめの検査が、特別扱いしない code では従来どおりサーバーの `detail` が出ることを固定するので、分岐が他の code まで巻き込んでいないことも分かれる。
- **Change-Resistance Results**:
  リスクは `low` のため必須ではないが、代表的な誤実装を 2 つ実測した。backend 側で 409 の分岐を元の 403 `mfa_enrollment_not_allowed` へ戻すと、受け入れ検査の 2 つの部分試験が落ち、3 つめ (403 のままであるべき側) は通ったままだった。frontend 側で `enrollmentErrorMessage` の code 分岐を外して常に `cause.message` を返すと、`30 pass` が `29 pass 1 fail` になった。
- **Verification Results**:
  - `mise run verify` - passed (exit 0)
  - `mise run check-spec` - ok (148 document(s), 333 operation(s), 845 TypeSpec symbol(s))
  - `mise run check-api-compat` - `no breaking changes` (409 の追加は非破壊。ベースラインは更新していない)
  - `mise run test-go-package -- ./backend/oauth2/handlers_http/...` - ok
  - `mise run test-ui-unit-file -- src/features/auth-flow/AuthFlowPages.test.tsx` - 30 pass, 0 fail
  - `mise run spec-diff` - `added TypeSpec declarations: MfaAlreadyEnrolledError`
