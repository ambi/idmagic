---
depends_on: []
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-08
priority: p2
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: 消去要求を受けた利用者の WebAuthn 資格情報とリカバリコードが、これまで残っていたのに消えるようになる。保管されるデータの範囲が変わるので、運用と法務の読み手に見える。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-513.md }
affected_spec:
  - { path: docs/standards.md, requirement: GDPR-ERASURE }
primary_use_cases:
  - id: purge-destroys-every-authentication-credential
    requirement: GDPR-ERASURE
    observable_result: 消去要求を受けた利用者の WebAuthn 資格情報とリカバリコードが、どの読み出し経路からも返らなくなる。
    unit_test:
      {
        path: backend/authentication/usecases/credential_erasure_standards_test.go,
        name: TestCredentialErasureLeavesNothingAuthenticable,
        task: test-go-race,
      }
    e2e_test:
      {
        path: backend/idmanagement/user/handlers_http/purge_credential_cascade_test.go,
        name: TestAdminUserAPIPurgeDestroysWebAuthnCredentialsAndRecoveryCodes,
        task: test-go-race,
      }
    unit_fault_model: cascadeDeleteForSub が WebAuthn 資格情報の DeleteAllForSub を呼ばない。
    e2e_fault_model: adminUserDeps が受け取った 2 つの port を AdminUserDeps へ渡さない（配線の欠け）。
initial_context:
  specification:
    - docs/standards.md#GDPR-ERASURE
    - docs/contexts/identity-management/states.md
  typespec: []
  source:
    - backend/idmanagement/user/usecases/admin_users.go
    - backend/idmanagement/user/handlers_http/admin_user_handler.go
    - backend/idmanagement/deps_http/deps.go
    - backend/shared/http/server_http/routes.go
    - backend/authentication/webauthn/ports/webauthn_credential_repository.go
    - backend/authentication/webauthn/db_memory/webauthn.go
    - backend/authentication/webauthn/domain/webauthn_credential.go
    - backend/authentication/webauthn/usecases/verify_webauthn_factor.go
    - backend/authentication/recovery/ports/recovery_code_repository.go
    - backend/authentication/recovery/db_memory/recovery_code.go
    - backend/authentication/recovery/domain/recovery_code.go
    - backend/authentication/recovery/usecases/recovery_codes.go
    - backend/cmd/internal/bootstrap/memory.go
    - backend/cmd/internal/bootstrap/postgres.go
    - infra/schema/postgres.sql
  tests:
    - backend/authentication/usecases/credential_erasure_standards_test.go
    - backend/idmanagement/user/usecases/erasure_standards_test.go
    - backend/idmanagement/user/usecases/admin_users_test.go
    - backend/idmanagement/user/handlers_http/admin_user_handler_test.go
  stop_before_reading:
    - backend/oauth2
    - backend/saml
    - frontend
---

# Purge が WebAuthn 資格情報とリカバリコードを消さない

## Motivation

`docs/standards.md` の `GDPR-ERASURE` は「削除要求後は法的保存義務を除く PII を定義済み期間内に消去する。消去は IdManagement の UserLifecycle Purge 遷移と Authentication の資格情報破棄が個別に担う」と宣言している。

Purge の cascade は `backend/idmanagement/user/usecases/admin_users.go` の `cascadeDeleteForSub` にある。消しているのは Consent、リフレッシュトークン、セッション、パスワード履歴、MFA 要素、信頼済みデバイス、デバイスコード、承認要求の 8 種である。**`WebAuthnCredentialRepository` と `RecoveryCodeRepository` はここに現れない。** `AdminUserDeps` がその 2 つの port を持っていないので、配線の抜けではなく型の抜けである。両 port は `DeleteAllForSub` を「anonymize cascade から呼ばれる」と自ら doc コメントに書いているが、呼んでいるのは MFA の管理者リセット (`backend/authentication/mfa/usecases/admin_reset.go`) とリカバリコードの再発行 (`backend/authentication/recovery/usecases/recovery_codes.go`) だけである。

PostgreSQL 側の `recovery_codes` には `FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE` があるが、**Purge は users の行を削除せず Tombstone 化する UPDATE なので、この cascade は発火しない。** `webauthn_credentials` には外部キーそのものが無い。

結果として、消去要求を受けた利用者の WebAuthn 資格情報（credential id、公開鍵、AAGUID、ラベル、最終利用時刻）とリカバリコードのハッシュが、その利用者の id に紐付いたまま残る。

[[wi-502-back-cross-cutting-standards-rows-with-tests]] が `GDPR-ERASURE` にテストを対応付ける過程で見つけた。同項目はテストの追加と実装の修正を混ぜない方針なので、修正をここへ切り出した。同項目が足した `backend/authentication/usecases/credential_erasure_standards_test.go` は、この 2 種を観測に含めていない。

## Scope

- `AdminUserDeps` に WebAuthn 資格情報とリカバリコードの port を足し、`cascadeDeleteForSub` から消す。
- 本番の組み立てで両 port を配線する。実際の経路は `backend/shared/http/server_http/routes.go` の `idmhttp.Deps` と `backend/idmanagement/user/handlers_http/admin_user_handler.go` の `adminUserDeps` であり、`backend/cmd/internal/bootstrap` は既に両 repository を `Authentication` モジュールへ入れている。したがって足すのは中間の 2 層である。
- `backend/authentication/usecases/credential_erasure_standards_test.go` の観測へ 2 種を足す。同ファイルは `GDPR-ERASURE` を名指しているので、行の被覆がそのまま強くなる。
- 両 port の `DeleteAllForSub` の doc コメントが言う「anonymize cascade から呼ばれる」が、実際に成り立つようにする。

## Out of Scope

- `docs/standards.md` の `GDPR-ERASURE` の文面。宣言は正しく、満たしていないのは実装である。
- Tombstone 化をやめて users の行を物理削除する設計変更。外部キーの cascade に頼る形は、監査に必要な Tombstone と両立しない。
- `webauthn_credentials` に外部キーを足すこと。Purge が UPDATE である限り cascade は発火しないので、外部キーはこの欠陥を直さない。参照整合性そのものは別の関心である。
- 既存データの後追い消去。製品は未リリースなので、残留している行は存在しない。
- 監査記録の保持。[[wi-514-audit-retention-decision-and-sweep-disagree]] が持つ。
- `ProvisionFederatedUser` が組み立てる `AdminUserDeps`。生成と更新しか呼ばず cascade へ到達しないので、port を渡す必要が無い。

## Design

port を 2 つ足して cascade から呼ぶだけの変更である。新しい型も新しい判断も要らない。**要るのは、足した port が本番の組み立てで実際に届いていることの観測である。**

### 型と署名

```go
type AdminUserDeps struct {
    // ... 既存の 8 つの cascade 先
    WebAuthnCredentialRepo webauthnports.WebAuthnCredentialRepository
    RecoveryCodeRepo       recoveryports.RecoveryCodeRepository
}

func cascadeDeleteForSub(ctx context.Context, deps AdminUserDeps, sub string) error
```

`nil` を「未配線として何もしない」と扱う既存 8 port の形に合わせる。**合わせる代わりに、配線の抜けを型ではなくテストで捕まえる。** 型で強制する案（port を必須にして nil をエラーにする）は採らない。8 つの既存 port と扱いが分かれ、`ProvisionFederatedUser` のように cascade を通らない組み立てまで巻き込むためである。

効果の境界は既にすべて port にある。時刻は `DeleteUserInput.Now` から入り、cascade は削除しか行わないので、この変更で新しい効果は増えない。

### 配線の欠けをどこで捕まえるか

3 層ある。`backend/shared/http/server_http/routes.go` が `idmhttp.Deps` を組み、`backend/idmanagement/user/handlers_http/admin_user_handler.go` の `adminUserDeps` がそれを `AdminUserDeps` へ写し、`cascadeDeleteForSub` が使う。**use case を直接呼ぶテストは、真ん中の 2 層を 1 つも通らない。** そこだけを見ると、port を足して cascade から呼んだのに本番では常に `nil` という状態を、テストが緑のまま通す。

したがって観測を 2 つ置く。

| 観測 | 入口 | 捕まえるもの |
|---|---|---|
| Unit | `userusecases.DeleteUser` を直接呼ぶ | `cascadeDeleteForSub` が 2 種を消していない |
| Acceptance | `httpadapter.Register` が組んだ `DELETE /api/admin/v1/users/{sub}?purge=true` | `routes.go` か `adminUserDeps` の写し漏れ |

Acceptance の入口は既存の `newAdminUserHandler` が使う `httpadapter.Register` で、本番と同じ経路である。**[[wi-500-back-api-tokens-standards-rows-with-tests]] が自前で組んだスタックを測って存在しない欠陥を報告した失敗と同じ形を避ける。**

### 「消えた」をどう読むか

行の有無だけでは足りない。読むのは、その資格情報でもう認証が成立しないことである。

- **WebAuthn**: `ListBySub` が空であること。`BeginWebAuthnAssertion` と `FinishWebAuthnAssertion` はどちらも `len(stored) == 0` で `ErrWebAuthnNoCredential` を返すので、この述語がそのまま「この利用者では WebAuthn が成立しない」である。あわせて `FindByCredentialID` が nil を返すこと（credential id を知っていても引けない）。
- **リカバリコード**: `RecoveryCodeStatusFor` の `Total` と `Remaining` が 0 であること。これは製品自身が「この利用者に何本残っているか」を答える経路であり、0 は消費できるコードが 1 本も無いという意味である。

**消去前に同じ経路が値を返すことも読む。** 消去後だけを見るテストは、そもそも何も保存されていない状態と区別できない。リカバリコードは `GenerateRecoveryCodes` が返した平文を `ConsumeRecoveryCode` で 1 本消費し、実際に認証として成立することまで確かめてから消去する。

消去後に `ConsumeRecoveryCode` を呼んで拒否を読む形は採らない。同関数は先に `LoadSelfUser` を通り、Tombstone 化した利用者はそこで弾かれるので、拒否の理由がコードの不在なのか利用者の不在なのか区別できない。**片方の消滅がもう片方を覆い隠す観測になる。**

### 他人の資格情報を巻き込まないこと

`DeleteAllForSub` は sub で絞るが、消去の観測が 1 人分しかないと、全件削除する実装も同じように緑になる。消去しない 2 人目を置き、その資格情報が両方とも残ることを併せて読む。

## Plan

1. Acceptance の観測を先に書き、`?purge=true` のあとに 2 種が残っていることを確認する（Acceptance RED）。
2. Unit の観測を `credential_erasure_standards_test.go` へ足し、`DeleteUser` が 2 種を消していないことを確認する（Unit RED）。
3. `AdminUserDeps` に port を 2 つ足し、`cascadeDeleteForSub` から `DeleteAllForSub` を呼ぶ。Unit が緑になる。
4. `deps_http.Deps` と `adminUserDeps` と `routes.go` を通して配線する。Acceptance が緑になる。
5. 両 port の doc コメントを、実際の呼び出し元と合うように直す。
6. 故障注入で、cascade の各呼び出しと配線の各層を 1 つずつ崩して落ちることを確かめる。

## Tasks

- [x] T001 [Acceptance] `DELETE /api/admin/v1/users/{sub}?purge=true` を入口に、2 種が消えることの観測を書く。
  `TestAdminUserAPIPurgeDestroysWebAuthnCredentialsAndRecoveryCodes`
  （`backend/idmanagement/user/handlers_http/purge_credential_cascade_test.go`）。既存の
  `newAdminUserHandler` へ `httpadapter.Deps` を触る可変長オプションを足し、2 つの port を
  足せるようにした。既存の呼び出しは 1 つも変わっていない。
  recipe: `mise run test-go-package -- ./backend/idmanagement/user/handlers_http`
- [x] T002 [Unit] `credential_erasure_standards_test.go` の観測へ 2 種を足す。
  `TestCredentialErasureLeavesNothingAuthenticable`。WebAuthn は `ListBySub` と
  `FindByCredentialID`、リカバリコードは `GenerateRecoveryCodes` が返した平文を
  `ConsumeRecoveryCode` で 1 本消費してから `RecoveryCodeStatusFor` で残数を読む。
  recipe: `mise run test-go-package -- ./backend/authentication/usecases`
- [x] T003 [Use Cases] `AdminUserDeps` に port を足し、`cascadeDeleteForSub` から消す。
  既存 8 port と同じく nil を未配線として扱う。
  recipe: `mise run test-go-package -- ./backend/authentication/usecases`
- [x] T004 [Adapters] `deps_http.Deps`、`adminUserDeps`、`routes.go` を通して配線する。
  `backend/cmd/internal/bootstrap` は既に両 repository を `Authentication` モジュールへ
  入れていたので、足したのは中間の 2 層だけである。
  recipe: `mise run test-go-package -- ./backend/idmanagement/user/handlers_http`
- [x] T005 [Docs] 両 port の `DeleteAllForSub` の doc コメントを実際の呼び出し元に合わせる。
  呼び出し元は WebAuthn が Purge と管理者リセットの 2 つ、リカバリコードが本人による失効、
  管理者リセット、Purge の 3 つである。
  recipe: `mise run lint-go`
- [x] T006 [Evidence] 宣言した 2 つの故障を注入し、対応する観測が落ちることを記録する。
  5 件注入して 5 件とも検出した。内訳は Change-Resistance Results。
  recipe: `mise run test-go-changed`
- [x] T007 [Verify] `mise run verify`。

## Verification

- Purge を経た利用者について、WebAuthn 資格情報とリカバリコードがどちらも読み出せない。
- 消去前は同じ経路でどちらも読み出せ、リカバリコードは実際に 1 本消費できる。
- 消去しなかった別の利用者の資格情報は両方とも残る。
- `backend/authentication/usecases/credential_erasure_standards_test.go` の観測に 2 種が入り、`cascadeDeleteForSub` からそれぞれの呼び出しを外すとそのテストが落ちる。
- 本番の配線（`routes.go` と `adminUserDeps`）を 1 層ずつ外すと、Acceptance の観測が落ちる。
- `mise run verify`

## Completion

- **Completed At**: 2026-09-08
- **Summary**:
  Purge の cascade が破棄する資格情報に、WebAuthn 資格情報とリカバリコードの 2 種が加わった。
  これで `docs/standards.md` の `GDPR-ERASURE` が言う「Authentication の資格情報破棄」に、
  Authentication が持つ資格情報の取りこぼしが無くなる。
  `mise run spec-diff` は規範差分なしであり、本項目は既存の `GDPR-ERASURE` を実装で満たす。
  変更は 3 層に分かれる。`AdminUserDeps` へ port を 2 つ足して `cascadeDeleteForSub` から
  `DeleteAllForSub` を呼び、`deps_http.Deps` と `adminUserDeps` と `routes.go` でそこへ配線し、
  両 port の doc コメントを実際の呼び出し元に合わせた。`backend/cmd/internal/bootstrap` は
  既に両 repository を `Authentication` モジュールへ入れていたので、足したのは中間の 2 層である。
  **観測を 2 つに分けたことが効いた。** use case を直接呼ぶ観測は、配線の 2 層を 1 つも通らない。
  実際、`adminUserDeps` から port を落とす故障も `routes.go` から落とす故障も、use case 側の
  テストは緑のまま通した。管理 API を入口にした観測だけが両方を捕まえている。
- **Acceptance RED Evidence**:
  - **Test**: `TestAdminUserAPIPurgeDestroysWebAuthnCredentialsAndRecoveryCodes`
    (`backend/idmanagement/user/handlers_http/purge_credential_cascade_test.go`)
  - **Requirement**: N/A: 観測は docs/standards.md の GDPR-ERASURE 行に対応し、REQ 番号を持つ規範シナリオには対応しない。
  - **Observed Failure**: `消去後に残った資格情報 webauthn=1 recovery=2, want どちらも 0`。
    `DELETE /api/admin/v1/users/{sub}?purge=true` が 204 を返したあとも、その利用者の
    WebAuthn 資格情報 1 件とリカバリコード 2 件が引けたままだった。
  - **Detection Reason**: 観測は本番と同じ `httpadapter.Register` が組んだ経路を通り、消去後に
    Authentication 側の repository を読み直す。204 という応答だけを読むテストは、消去を宣言して
    何も消さない実装をそのまま通す。
- **Unit RED Evidence**:
  - **Test**: `TestCredentialErasureLeavesNothingAuthenticable`
    (`backend/authentication/usecases/credential_erasure_standards_test.go`)
  - **Requirement**: N/A: 観測は docs/standards.md の GDPR-ERASURE 行に対応し、REQ 番号を持つ規範シナリオには対応しない。
  - **Observed Failure**: `消去後に残った WebAuthn 資格情報=[0x...] err=<nil>`。
    port を型に足しただけで `cascadeDeleteForSub` から呼ばない状態で観測した。それより前は
    `AdminUserDeps` に該当のフィールドが無く、テストがコンパイルできない
    (`unknown field WebAuthnCredentialRepo in struct literal`)。**型の欠けそのものが欠陥なので、
    記録する RED は型を足したあとの表明の失敗にした。**
  - **Detection Reason**: 消去前に同じ経路が値を返すことも読むので、そもそも何も保存されて
    いない状態と区別できる。リカバリコードは消去前に実際に 1 本消費しており、消去後の
    `remaining=9` がその消費を裏づけている。
- **Primary Use Case Evidence**:
  - id: purge-destroys-every-authentication-credential
    unit_red: "TestCredentialErasureLeavesNothingAuthenticable は、port を型に足しただけで cascade から呼ばない状態で、消去後に残った WebAuthn 資格情報=[0x...] を得て落ちた。それより前は AdminUserDeps に該当のフィールドが無く、unknown field WebAuthnCredentialRepo in struct literal でコンパイルできなかった。"
    e2e_red: "TestAdminUserAPIPurgeDestroysWebAuthnCredentialsAndRecoveryCodes は、DELETE /api/admin/v1/users/{sub}?purge=true が 204 を返したあとも 消去後に残った資格情報 webauthn=1 recovery=2 を得て落ちた。"
    unit_fault_injection: "cascadeDeleteForSub から WebAuthn の DeleteAllForSub の呼び出しを外すと、同テストが同じ観測で落ちた。リカバリコード側を外すと total=10 remaining=9 で落ちた。"
    e2e_fault_injection: "adminUserDeps が 2 つの port を AdminUserDeps へ渡さないようにすると、同テストが webauthn=1 recovery=2 で落ちた。**このとき unit のテストは緑のまま通った。** routes.go から idmhttp.Deps への受け渡しを外した場合も同じ観測になる。"
- **Change-Resistance Results**:
  宣言した 2 つの故障に、配線の各層と絞り込みの向きを足して 5 件注入し、5 件とも検出した。
  等価変異は無い。故障は注入のたびに元へ戻している。

  | 故障 | 落ちたテストと観測 |
  |---|---|
  | `cascadeDeleteForSub` が WebAuthn の `DeleteAllForSub` を呼ばない（宣言した unit_fault_model） | `TestCredentialErasureLeavesNothingAuthenticable`: 消去後に資格情報が残る |
  | `cascadeDeleteForSub` がリカバリコードの `DeleteAllForSub` を呼ばない | 同テスト: `total=10 remaining=9` |
  | `adminUserDeps` が 2 つの port を `AdminUserDeps` へ渡さない（宣言した e2e_fault_model） | `TestAdminUserAPIPurgeDestroysWebAuthnCredentialsAndRecoveryCodes`: `webauthn=1 recovery=2`。**use case 側のテストは緑のまま通った。** |
  | `routes.go` が 2 つの port を `idmhttp.Deps` へ渡さない | 同テスト: `webauthn=1 recovery=2` |
  | WebAuthn の `DeleteAllForSub` が sub で絞らず全件消す | 同テスト: `対照の利用者の資格情報が消えた webauthn=0` |
- **Verification Results**:
  - `mise run test-go-package -- ./backend/authentication/usecases` - passed
  - `mise run test-go-package -- ./backend/idmanagement/user/handlers_http` - passed
  - `mise run test-go-package -- ./backend/idmanagement/user/usecases` - passed
  - `mise run test-go-changed` - passed
  - `mise run lint-go` / `mise run format-go` - passed
  - `mise run test-ui-e2e` - `N/A: 変更は Go の cascade と配線だけで、ブラウザーへ到達する経路が無い。`
  - `mise run test-go-race` - passed（`verify` の中で完走し、失敗パッケージは 0）
  - `mise run check-spec` - passed（171 正準文書、156 規範、311 規則、744 例）
  - `mise run check-api-compat` - passed（破壊的変更なし）
  - `mise run check-ids` - passed（513 件）
  - `mise run check-work-items` - passed（513 件）
  - `mise run spec-diff` - passed（規範差分なし）
  - `mise run verify` - passed。初回は無関係な `AccountDataPage` の UI 単体試験が 1 件失敗したが、対象ファイルの 5 件は単独再実行で成功し、全体の再実行でも 683 件すべてが成功した。

## Risk Notes

- **port を足しただけで配線を忘れる。** `AdminUserDeps` の他の port と同じく `nil` を「未配線として何もしない」と扱うので、本番の組み立てで渡し忘れても use case のテストは落ちない。`httpadapter.Register` を入口にした観測を置き、配線の各層を外して落ちることを確かめる。
- **消したことを行の有無だけで観測する。** WebAuthn 資格情報は公開鍵と credential id を持つので、行が消えたことに加えて、製品が「この利用者では WebAuthn が成立しない」と判断する述語まで読む。
- **利用者の消滅が資格情報の消滅を覆い隠す。** Tombstone 化した利用者はどの自己操作の入口でも先に弾かれる。消去後の観測は、利用者の状態を見ない経路（repository と `RecoveryCodeStatusFor`）に置く。
- **リカバリコードの外部キーに引きずられる。** `ON DELETE CASCADE` があるので「消える」と読める。Purge は UPDATE なので発火しない。テストは Purge を通した観測にする。
