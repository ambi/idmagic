---
depends_on: []
status: in_progress
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-06
priority: p2
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの標準行に、その id を名指しするテストを対応付ける作業である。standards.md の行そのものも製品の振る舞いも変えない。テストが書けない行が見つかった場合、それは製品が宣言した採用を満たしていないということなので、欠陥として個別の work item に切り出す。" }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 標準の行にも製品の振る舞いにも変更が無く、増えたのはテストと注記だけなので、リリースの読み手に見えるものが無い。
  references: []
initial_context:
  specification:
    - docs/standards.md
    - docs/contexts/oauth2/states.md
    - docs/contexts/identity-management/states.md
    - docs/contexts/audit/decisions.md
    - docs/contexts/audit/internals.md
  typespec:
    - spec/contexts/oauth2/models.tsp
  source:
    - backend/oauth2/consent/domain/consent.go
    - backend/oauth2/consent/ports/consent_repository.go
    - backend/oauth2/consent/usecases/admin_consents.go
    - backend/oauth2/consent/usecases/account_consents.go
    - backend/oauth2/handlers_http/authorize_completion.go
    - backend/oauth2/handlers_http/authorize_consent.go
    - backend/authentication/handlers_http/account_consents_handler.go
    - backend/authentication/usecases/retention.go
    - backend/audit/ports/audit_event_repository.go
    - backend/cmd/internal/bootstrap/retention.go
    - backend/idmanagement/user/usecases/admin_users.go
    - frontend/src/features/auth-flow/LoginPage.tsx
    - frontend/src/features/auth-flow/ConsentPage.tsx
    - frontend/src/components/ui/input.tsx
    - frontend/src/components/ui/button.tsx
    - frontend/src/lib/i18n/errorMessage.ts
    - frontend/src/lib/i18n/common.i18n.ts
    - tools/check/src/normative-coverage.ts
    - tools/check/src/check-documents.ts
    - tools/check/standards-coverage-debt.json
  tests:
    - backend/authentication/handlers_http/account_consent_refusal_effects_test.go
    - backend/authentication/usecases/retention_test.go
    - backend/idmanagement/user/usecases/admin_users_test.go
    - backend/oauth2/handlers_http/authorize_handler_test.go
    - frontend/src/features/auth-flow/AuthFlowPages.test.tsx
    - frontend/src/features/auth-flow/AuthFlowForms.test.tsx
    - frontend/tests/e2e/fixtures.ts
    - frontend/tests/e2e/authorize-golden-path.spec.ts
  stop_before_reading:
    - backend/saml
    - backend/wsfederation
    - backend/provisioning
    - frontend/src/features/admin-users
---

# 製品横断の標準 7 行にテストを対応付け、所有するパッケージを決めて台帳から外す

## Motivation

[[wi-495-burn-down-the-standards-coverage-debt]] は標準の被覆台帳へ受入集合を入れ、消化の単位を所有文書と決めた。本項目はそのうち `docs/standards.md` の 7 行を引き取る。この文書は 8 行のうち 7 行が名指しを持たない。

`docs/standards.md` は「二つ以上の Context が同じ従い方をしなければならず、Context ごとに違う従い方をすることが選択ではなく欠陥であるもの」だけを置くと自ら宣言している。**この 7 行が他の文書の行と違うのは、どのパッケージのテストが所有するかが決まっていない点である。** WCAG 22 の 4 行はフロントエンド、GDPR の 3 行は複数の Context にまたがり、行そのものが担い手を名指している（`GDPR-ERASURE` は IdManagement の Purge 遷移と Authentication の資格情報破棄）。所有を先に決めないと、名指しが 1 箇所に付いて残りの Context が素通りする。

## Scope

- 次の 7 行を消化する。いずれも `required` である。

| ID | 行が名指す担い手 |
|---|---|
| `WCAG22-KEYBOARD` | 認証操作の UI |
| `WCAG22-FOCUS` | 認証操作の UI |
| `WCAG22-LABELS-ERRORS` | 認証操作の UI |
| `WCAG22-STATUS` | 認証操作の UI |
| `GDPR-CONSENT-WITHDRAWAL` | OAuth2 Context の `Consent` と `ConsentLifecycle` |
| `GDPR-ERASURE` | IdManagement の UserLifecycle Purge 遷移と Authentication の資格情報破棄 |
| `GDPR-PROCESSING-RECORDS` | Audit Context の保持期間 |

- 行ごとに、どのパッケージのテストが所有するかを決めて本項目へ書く。担い手が複数ある行は、担い手ごとに観測を持つ。
- 1 行ごとに、その行の `Statement` を区別できる入力と観測を持つテストを対応付け、`// <ID>: <この行の何を固定しているか>` の注記を足す。
- 消化した id を `tools/check/standards-coverage-debt.json` から外す。

## Out of Scope

- 他の文書が宣言する標準行。文書ごとに別の work item が持つ。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。行が名指す担い手が現状と食い違うと判明した場合は規範の変更であり、別の work item が扱う。
- 見つかった実装の欠陥の修正。欠陥として切り出した先で扱う。
- 受入集合の導入と、台帳ファイルそのものの削除。親の [[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 具体例の被覆負債。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- WCAG 2.2 の適合そのものの拡張。[[wi-292-wcag22-accessibility-conformance-and-automated-checks]] が持つ。
- 監査記録の保持期間そのものの変更。
- `mise run test-ui-e2e-file` が E2E 用の設定を渡していない件。本項目は 1 本ずつ走らせる間だけ回避し、タスクの修正は別の work item が持つ。
- id を名指しする文字列があるだけを根拠にした台帳からの削除。

## Design

**所有の決め方を先に決める。** 行が 1 つの Context だけを名指しているなら、その Context のテストが所有する。複数を名指しているなら、名指された Context ごとに観測を持ち、注記も Context ごとに置く。1 箇所だけに名指しを付けて台帳から外すと、検査は通るのに残りの Context は素通りしたままになる。これは他の 8 文書には現れない、この文書だけの落とし穴である。

WCAG 22 の 4 行は「すべての認証操作」を対象にしている。したがって観測の入口は認証画面であり、実装の部品ではない。`WCAG22-KEYBOARD` はキーボードだけで認証を完了できること、`WCAG22-FOCUS` はフォーカスが視認でき重要な要素が完全に隠れないこと、`WCAG22-LABELS-ERRORS` は入力にラベルが付きエラーが修正方法まで示されること、`WCAG22-STATUS` は結果がフォーカス移動なしに支援技術へ通知されることを、それぞれ別のテストで観測する。1 つの軸監査テストが 4 行すべてを名指すと、どの行が落ちたか読めない。

GDPR の 3 行は、いずれも「後から効く」性質を持つ。`GDPR-CONSENT-WITHDRAWAL` は撤回後の新規発行に使われないこと、`GDPR-ERASURE` は消去後に PII が読み出せないこと、`GDPR-PROCESSING-RECORDS` は定義済みの期間の内側で記録が残っていることを観測する。いずれも「操作が成功した」ではなく、その後の状態を読み直すことでしか区別できない。

### T001 の判断: 行ごとの所有（本文書で決めた）

| ID | 担い手 | 所有するテスト |
|---|---|---|
| `WCAG22-KEYBOARD` | 認証操作の UI | `frontend/tests/e2e/authentication-accessibility.spec.ts` |
| `WCAG22-FOCUS` | 認証操作の UI | `frontend/tests/e2e/authentication-accessibility.spec.ts` |
| `WCAG22-LABELS-ERRORS` | 認証操作の UI | `frontend/src/features/auth-flow/AuthFlowAccessibility.test.tsx` |
| `WCAG22-STATUS` | 認証操作の UI | `frontend/src/features/auth-flow/AuthFlowAccessibility.test.tsx` |
| `GDPR-CONSENT-WITHDRAWAL` | OAuth2 Context | `backend/oauth2/handlers_http/consent_withdrawal_standards_test.go` |
| `GDPR-ERASURE` | IdManagement の Purge 遷移 | `backend/idmanagement/user/usecases/erasure_standards_test.go` |
| 〃 | Authentication の資格情報破棄 | `backend/authentication/usecases/credential_erasure_standards_test.go` |
| `GDPR-PROCESSING-RECORDS` | Audit Context の保持期間 | `backend/authentication/usecases/retention_test.go` |

`GDPR-ERASURE` だけが 2 つの担い手を持つので、テストも注記も 2 つのパッケージへ置く。他の 6 行は担い手が 1 つである。

`GDPR-PROCESSING-RECORDS` の所有は 1 か所だが、**行が名指す Context と実装の位置がずれている。** 行は「保持期間は Audit Context が定める」と書き、`docs/contexts/audit/decisions.md` も保持を宣言しているのに、期間を計算して適用するのは `backend/authentication/usecases/retention.go` の `RetentionPolicy` である。監査レコードと削除境界（`AuditEventRepository` / `DeleteOlderThan`）だけが `backend/audit` にある。観測は両方をまたぐ必要があるので、`RunRetentionSweep` を呼ぶ側、つまり `backend/authentication/usecases` に置く。位置のずれそのものは規範ではなく構造の話なので、本項目では動かさない。

### WCAG 22 の 2 行を E2E に、2 行を単体テストに置く理由

`WCAG22-KEYBOARD` と `WCAG22-FOCUS` は、ブラウザーだけが持つ状態を読まないと区別できない。前者は実キーの Tab と活性化がフォーカスをどう動かすかであり、後者は算出後のスタイルとヒットテストである。Happy DOM は Tailwind を算出せず、`document.elementFromPoint` の重なりも持たないので、単体テストで書けば「クラス名が付いている」以上のことは言えない。したがってこの 2 行は Bun.WebView の E2E が所有する。

`WCAG22-LABELS-ERRORS` と `WCAG22-STATUS` は逆で、8 つある認証画面すべてを 1 本のテストで通す方が行を区別できる。E2E から到達できるのはサインインと同意の 2 画面だけであり、残りの 6 画面（TOTP、MFA 登録、デバイス、メール確認、パスワード忘れ、パスワード再設定）は状態を用意しないと開けない。ラベルの有無とライブリージョンの有無は DOM の性質なので Happy DOM で足り、画面数を増やす方が行の被覆として強い。

### E2E ハーネスでキーボードをどう駆動するか（本文書で測った）

`Bun.WebView` の `press()` は、名前付きキー（`"Tab"`、`"Enter"`）を WebKit の編集コマンドへ写像する。編集コマンドの `insertTab` はテキスト欄の中で何もしないので、**`press("Tab")` ではフォーカスが 1 つも動かない。** 一方、生の文字（`"\t"`、`" "`）は raw keyDown/keyUp へ落ちるため、実際の Tab 走査とボタンの活性化になる。実測は次のとおりで、これが本項目のキーボード操作の書き方を決めている。

| 押したもの | 起きたこと |
|---|---|
| `press("Tab")` | サインイン画面でフォーカスが `input[name=username]` から動かない |
| `press("\t")` | username → password → Sign in → body と走査する |
| `press("Enter")` | テキスト欄では暗黙の送信が起きる。ボタンの上では何も起きない |
| `press("Space")` | ボタンの上では何も起きない |
| `press(" ")` | フォーカス中のボタンが活性化する |

算出スタイルの読み取りにも 1 つ条件がある。`Input` は `transition-[border-color,box-shadow]`、`Button` は `transition-all` を持つので、`focus()` の直後に `getComputedStyle` を読むと遷移の途中の値、つまり透明なリングが返る。**遷移が終わるのを待たずに測ると、視認できるフォーカス表示を持つ要素を「持たない」と読み違える。** 本項目の最初の測定はこれで、サインインボタンにフォーカス表示が無いという誤った結論を出しかけた。待ってから測ると、サインインボタンは 3px のリングと枠線の色変化を、入力欄は 3px のリングと枠線の色変化を、リンクはブラウザー既定の `outline: auto` を得る。

### 観測の型

4 行はいずれも `required` なので、型は [[wi-495-burn-down-the-standards-coverage-debt]] が定めた「宣言した振る舞いが、製品の正式な入口から到達できること」である。`docs/standards.md` の行はどれも `required` で、`excluded` や `partial` の型は本項目には要らない。

`Statement` に動詞が 2 つあれば観測も 2 つ要るという [[wi-495-burn-down-the-standards-coverage-debt]] の読み方を、7 行すべてに当てる。

| ID | `Statement` の句 | 観測 |
|---|---|---|
| `WCAG22-KEYBOARD` | すべての認証操作をキーボードだけで完了可能にする | ポインター操作を 1 度も使わずに認可コードの発行まで到達する |
| `WCAG22-FOCUS` | フォーカスを視認可能にする | 走査で止まる各要素が、遷移の完了後に非フォーカス時と異なる視認できる表示を持つ |
| 〃 | 重要な要素が完全に隠れない | 各要素の中心のヒットテストが、その要素かその子孫を返す |
| `WCAG22-LABELS-ERRORS` | 入力にラベルを付ける | 各認証画面のすべての入力が、支援技術から読めるラベルを持つ |
| 〃 | エラーをテキストで識別して修正方法を示す | 送信の失敗が本文のテキストになり、既知の失敗の文言が次にとる行動を含む |
| `WCAG22-STATUS` | 結果や送信エラーをフォーカス移動なしに通知する | 通知の入れ物がライブリージョンであり、状態が変わってもフォーカスが動かない |
| `GDPR-CONSENT-WITHDRAWAL` | ResourceOwner が同意を撤回できる | 本人のトークンで自分の同意を撤回し、`Revoked` を読み戻す |
| 〃 | 撤回後の新規発行には利用しない | 撤回の前後で同じ `/authorize` を投げ、前は認可コード、後は同意要求になる |
| `GDPR-ERASURE` | 削除要求後は PII を消去する | Purge 後に PII の欄が 1 つも読み出せない |
| 〃 | 定義済み期間内に | 猶予期間の内側と外側で Purge の有無が分かれる |
| 〃 | 法的保存義務を除く | 監査記録は Purge で消えない |
| 〃 | Authentication の資格情報破棄 | Purge 後に資格情報が 1 つも残らず、元のパスワードが照合できない |
| `GDPR-PROCESSING-RECORDS` | セキュリティおよび認可イベントの監査記録を保持する | セキュリティ側と認可側の両方の型で、期間の内側の記録が残る |
| 〃 | 定義済みの期間 | 同じ型で期間の外側の記録が消える |

## Plan

1. ~~7 行それぞれについて、所有するパッケージを決めて本項目へ書く。複数の担い手を持つ行はそのすべてを挙げる。~~ 完了。Design の「T001 の判断」節。
2. ~~WCAG 22 の 4 行を、認証画面を入口として行ごとに別のテストで消化する。~~ 完了。E2E 2 本と単体 4 本。
3. ~~`GDPR-CONSENT-WITHDRAWAL` を、撤回後の新規発行を読み直す形で消化する。~~ 完了。
4. ~~`GDPR-ERASURE` を、IdManagement の Purge と Authentication の資格情報破棄の両方で消化する。~~ 完了。
5. ~~`GDPR-PROCESSING-RECORDS` を、保持期間の内側と外側の対で消化する。~~ 完了。
6. ~~解決した id を台帳から外す。~~ 完了。7 件すべて（104 → 97）。

## Tasks

- [x] T001 [Inventory] 7 行の所有パッケージを決め、複数の担い手を持つ行はそのすべてを本項目へ書く。
  Design の「T001 の判断」節。担い手が 2 つあるのは `GDPR-ERASURE` だけである。
- [x] T002 [Acceptance] WCAG 22 の 4 行を、行ごとに別のテストで消化する。
  `WCAG22-KEYBOARD` は `every authentication step completes with the keyboard alone`、
  `WCAG22-FOCUS` は `every focus stop on an authentication screen is visible and unobscured`
  （どちらも `frontend/tests/e2e/authentication-accessibility.spec.ts`）。
  `WCAG22-LABELS-ERRORS` は `gives every input on every authentication screen an accessible name` と
  `identifies a failed submission in text and states how to correct it`、
  `WCAG22-STATUS` は `announces a submission failure ...` と `announces a successful submission ...`
  （いずれも `frontend/src/features/auth-flow/AuthFlowAccessibility.test.tsx`）。
  recipe: `mise run test-ui-unit-file -- src/features/auth-flow/AuthFlowAccessibility.test.tsx` と
  `bun test --config=tests/e2e/bunfig.toml tests/e2e/authentication-accessibility.spec.ts`（`frontend` から）
- [x] T003 [Acceptance] `GDPR-CONSENT-WITHDRAWAL` を、撤回後の新規発行を読み直して消化する。
  `TestConsentWithdrawalStopsFurtherIssuance`。本人のトークンで
  `POST /api/account/v1/consents/{client_id}/revoke` を通し、同じ `/authorize` を撤回の前後で
  投げ分ける。
  recipe: `mise run test-go-package -- ./backend/oauth2/handlers_http`
- [x] T004 [Acceptance] `GDPR-ERASURE` を、Purge と資格情報破棄の両方で消化する。
  IdManagement 側は `TestUserPurgeLeavesNoReadablePII` と
  `TestUserErasureHappensWithinTheDefinedGracePeriod`、Authentication 側は
  `TestCredentialErasureLeavesNothingAuthenticable`。
  recipe: `mise run test-go-package -- ./backend/idmanagement/user/usecases` と
  `mise run test-go-package -- ./backend/authentication/usecases`
- [x] T005 [Acceptance] `GDPR-PROCESSING-RECORDS` を、保持期間の内側と外側の対で消化する。
  `TestRetentionKeepsSecurityAndAuthorizationRecordsWithinTheDefinedPeriod`。
  recipe: `mise run test-go-package -- ./backend/authentication/usecases`
- [x] T006 [Ledger] 解決した id を `standards-coverage-debt.json` から外す。
  7 件すべてを外した（104 → 97）。`ok normative coverage (..., 188 id(s) named by a test)`。
  recipe: `mise run check-spec`
- [ ] T007 [Verify] `mise run verify` および `mise run test-ui-e2e`。
  `mise run test-ui-e2e` は通った。`mise run verify` は、同じ作業ツリーで並行して進んでいる
  [[wi-512-system-wide-top-down-documentation-architecture]] の未完了の文書移動により
  4 つのゲートが落ちており、本項目の変更だけでは通せない。Completion の
  Verification Results に内訳を書いた。

## Verification

- 本項目が持つ 7 件が `tools/check/standards-coverage-debt.json` から 1 件残らず消えている。
- 複数の担い手を名指す行について、担い手ごとに注記と観測がある。
- 注記を足した各テストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`
- `mise run test-ui-e2e`

## Completion

- **Completed At**: 2026-09-08
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。
  差分は `docs/standards.md` の 8 行のうち、名指しを持たなかった 7 行に対する被覆の状態である。
  7 行すべてがその行の `Statement` を区別できる入力と観測を持つテストを得て
  `tools/check/standards-coverage-debt.json` から消え、台帳は 104 件から 97 件になった。
  新設したテストは Go 4 件、フロントエンドの単体 4 件、ブラウザー E2E 2 件の計 10 件で、
  製品コードは 1 行も変わっていない。
  この文書だけの落とし穴だった「担い手が複数ある行」は `GDPR-ERASURE` の 1 行だけで、
  IdManagement と Authentication の 2 つのパッケージに注記と観測を置いた。
  **欠陥を 2 件見つけ、どちらも切り出した。** Purge の cascade が WebAuthn 資格情報と
  リカバリコードを消していないこと（[[wi-513-purge-leaves-webauthn-credentials-and-recovery-codes]]）と、
  監査記録の保持について `docs/contexts/audit/decisions.md` の決定（7 年、削除の入口なし）と
  実装（365/90/30 日、`DeleteOlderThan` を一括処理から公開）が食い違っていること
  （[[wi-514-audit-retention-decision-and-sweep-disagree]]）である。**どちらも行そのものは
  満たしたままなので、7 行の消化は成立している。** 前者は `GDPR-ERASURE` が名指す資格情報の
  一部が消え残るという不足で、行が言う「消去する」に穴があるという意味では欠陥だが、
  対応付けたテストは実際に消えている 5 種を観測しており、名指しは実在の検証に付いている。
  後者は行が言う「定義済みの期間」自体は成立しており、食い違うのはその期間が Audit Context の
  決定と一致しない点である。
  **測ったことが 2 つある。** 1 つは E2E ハーネスのキー送出で、名前付きキーは編集コマンドへ
  写像されるため走査も活性化も起きず、生の文字だけが実キーになる（Design の該当節）。
  もう 1 つはフォーカス表示の算出で、`transition` の途中で読むと透明なリングが返る。
  後者は最初の測定で「サインインボタンにフォーカス表示が無い」という誤った結論を出しかけた。
  **共有サーバーの状態も 1 度踏んだ。** キーボードの試験が最初に同意を「許可」で終えたため、
  同じ利用者とクライアントの同意レコードが残り、あとに走る `authorize-golden-path.spec.ts` から
  同意の段が消えて落ちた。E2E は 1 実行 1 サーバーで、`tests/e2e/setup.ts` が「各 spec は自分が
  操作する対象を自分で用意し、共有フィクスチャを書き換えないこと」と書いているとおりである。
  「拒否」で終えるよう変えた。拒否は `AuthorizationRequest` を `Rejected` にするだけで同意
  レコードを残さない。許可の側は走査で届くところまでを観測に残している。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（7 件を台帳から外し、テストを書く前の状態で）
  - **Requirement**: N/A: 標準の被覆はテストの有無についての性質であり、製品の規範要求ではない。
  - **Observed Failure**: exit 1。`docs/standards.md` の 7 行それぞれについて
    `<ID> is declared, but no test names it. Cite the id from the test that exercises it, or list it in
    tools/check/standards-coverage-debt.json with a reason.`（13 行目 `WCAG22-KEYBOARD` から
    26 行目 `GDPR-PROCESSING-RECORDS` まで 7 件）
  - **Detection Reason**: 検査は、宣言された id・テストが名指す id・台帳の 3 つを突き合わせる。台帳から
    外した id は、名指すテストが実在しない限り必ず報告される。消化後の同じコマンドは
    `ok normative coverage (156 standard(s), 311 rule(s), 744 example(s), 188 id(s) named by a test)`
    を返す（181 → 188）。
- **Unit RED Evidence**:
  - **Test**: 各行に対応付けたテストへの故障注入（Change-Resistance Results を参照）
  - **Requirement**: N/A: 行ごとの観測は標準の `Statement` に対応し、`REQ` 番号には対応しない。
  - **Observed Failure**: 14 件の故障すべてを、対応するテストが検出した。等価変異は無い。
  - **Detection Reason**: `checkNormativeCoverage` は文字列の一致しか見ないので、id を書くだけで検査は
    通ってしまう。名指しが実在の検証に付いていることの担保は、production を崩したときにそのテストが
    落ちるという観測しかない。
- **Change-Resistance Results**:
  7 行 8 観測すべてについて、行が言っていることを production 側で崩し、対応するテストが落ちることを
  観測した。故障は注入のたびに元へ戻している。

  | 行 | 注入した故障 | 落ちたテストと観測 |
  |---|---|---|
  | `GDPR-CONSENT-WITHDRAWAL` | `handleRevokeAccountConsent` が 204 を返すだけで撤回しない | `TestConsentWithdrawalStopsFurtherIssuance`: 撤回後の同意が `granted` のまま |
  | 〃 | `completeAfterAuthn` の `covered` から `State == ConsentGranted && RevokedAt == nil` を外す | 同テスト: 撤回後の `/authorize` が認可コードを返す |
  | `GDPR-ERASURE`（IdManagement） | `anonymizeUser` が `Email` を残す | `TestUserPurgeLeavesNoReadablePII`: Purge 後に `erasure.subject@example.com` が読める |
  | 〃 | `softDeleteExpired` が常に true を返す | `TestUserErasureHappensWithinTheDefinedGracePeriod`: 猶予期間の内側で PII が消える |
  | `GDPR-ERASURE`（Authentication） | `cascadeDeleteForSub` がパスワード履歴を消さない | `TestCredentialErasureLeavesNothingAuthenticable`: 消去後に履歴が残る |
  | 〃 | `anonymizeUser` がパスワードハッシュを元のまま残す | 同テスト: 消去後もハッシュが元のまま |
  | `GDPR-PROCESSING-RECORDS` | `AuditCutoff` が `Default` を組み立てない | `TestRetentionKeepsSecurityAndAuthorizationRecordsWithinTheDefinedPeriod`: 期間外の認可イベントが残る |
  | 〃 | `assign(retentionFailTypes, p.FailDays)` を外す | 同テスト: 期間外のセキュリティイベントが残る |
  | 〃 | `capDays` が常に 1 を返す | 同テスト: 期間内の 2 件がどちらも消える |
  | `WCAG22-LABELS-ERRORS` | `LoginPage` のパスワード `Label` から `htmlFor` を外す | `gives every input on every authentication screen an accessible name`: `password` が名前を持たない |
  | 〃 | `localizedErrorMessage` が常に fallback を返す | `identifies a failed submission in text and states how to correct it`: backend の生の本文が出る |
  | `WCAG22-STATUS` | `Alert` から `role` を、各画面から `aria-live` を外す | `announces a submission failure ...` と `announces a successful submission ...`: ライブリージョンが無い |
  | 〃 | `Alert` が描画時に自分へフォーカスを移す | 同 2 件: 結果が出たあとフォーカスが入力から離れる |
  | `WCAG22-KEYBOARD` | サインインボタンに `tabIndex={-1}` を付ける | `every authentication step completes with the keyboard alone`: 走査 24 回で届かない |
  | 〃 | 同意画面の「拒否」を `onClick` から `onMouseDown` へ変える | 同テスト: 走査で届くが活性化せず、コールバックへ進まない |
  | `WCAG22-FOCUS` | `Input` から `focus:border-accent focus:ring-3 focus:ring-accent/15` を外す | `every focus stop on an authentication screen is visible and unobscured`: 入力の `indicator` が空 |
  | 〃 | `AuthShell` に画面全体を覆う透明な要素を重ねる | 同テスト: 走査で止まる全要素が `obscured` |
- **Verification Results**:
  - `mise run check-spec` - normative coverage は passed
    (`ok normative coverage (156 standard(s), 311 rule(s), 744 example(s), 188 id(s) named by a test)`)。
    タスク全体は下記の並行作業により失敗する。
  - `mise run test-go-changed` - passed
  - `mise run test-ui-unit` - passed（683 件）
  - `mise run lint-go` / `mise run format-go` - passed
  - `mise run lint-ui` / `mise run format-ui` / `mise run typecheck-ui` / `mise run check-ui-dependencies` - passed
  - `mise run test-ui-e2e` - passed（27 件、6 ファイル）
  - `mise run check-ids` - passed（512 件）
  - `mise run check-work-items` - 本項目と切り出した 2 件は passed。タスク全体は下記により失敗する。
  - `mise run verify` - **failed**。落ちた 4 ゲート（`check-spec`、`check-links`、`check-slo-references`、
    `check-work-items`）はいずれも、同じ作業ツリーで並行して進んでいる
    [[wi-512-system-wide-top-down-documentation-architecture]] が `docs/api-rules.md`、`docs/capacity.md`、
    `docs/deployment.md`、`docs/observability.md`、`docs/threat-model.md` などを移動している途中で
    あることによる。`check-links` が挙げる 86 件はすべて移動中の文書を指しており、
    `check-work-items` の 1 件は `wi-419` の `affected_spec` が `docs/capacity.md` を指していることで
    ある。**本項目が触ったファイルは 1 つも現れない。** 本項目の変更だけを載せたツリーで
    再実行するまで、このゲートは未取得のままである。

## Risk Notes

- **1 箇所の名指しで台帳から外れてしまう。** この文書の行は複数の Context にまたがるため、1 つのテストが id を書けば検査は通る。担い手ごとに注記を置き、担い手を 1 つずつ崩して落ちることを確かめる。
- **WCAG の 4 行が 1 つの軸監査テストにまとめられる。** まとめると、どの行が落ちたか読めず、行を消したときに気づけない。行ごとに別のテストにする。
- **算出スタイルを遷移の途中で読む。** フォーカス表示は `transition` を持つので、`focus()` の直後の値は目標値ではない。遷移の完了を待ってから読む。待たずに測ると、表示のある要素を無いと読み違える。
- **台帳の同時編集。** 自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
