---
depends_on: [wi-565-make-backing-declared-examples-cheap]
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: maintenance
affected_spec:
  - { path: docs/domain/signing-keys/scenarios.feature.md, requirement: REQ-SIGNINGKEYS-006 }
  - { path: spec/contexts/signing-keys/main.tsp, symbol: IdMagic.SigningKeys.Operations.RotateTenantSigningKey }
  - { path: spec/contexts/signing-keys/models.tsp, symbol: IdMagic.Contract.AdminRotateKeyRequest }
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: 管理者が XmlFederationSigning 鍵をローテートできるようになり、現在の署名鍵の無効化を拒否する 400 の問題コードが契約どおりの invalid_request へ変わる。どちらも API 利用者に見える変化である。未リリースのため移行の手順は要らない。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-552-back-signing-keys-examples-with-tests.md }
initial_context:
  specification:
    - docs/domain/signing-keys/scenarios.feature.md
    - docs/domain/signing-keys/decisions.md
  typespec:
    - IdMagic.SigningKeys.Operations.RotateTenantSigningKey
  source:
    - backend/signingkeys/handlers_http/admin_key_handler.go
    - backend/signingkeys/handlers_http/routes.go
    - backend/signingkeys/keys_memory/key_store.go
    - backend/signingkeys/keys_vault/vault_key_store.go
    - backend/signingkeys/usecases/archive_expired_signing_keys.go
    - backend/signingkeys/usecases/tenant_key_health.go
    - backend/wsfederation/tokens_saml/signer_provider.go
    - backend/cmd/idmagic-batch/main.go
    - backend/shared/http/testing_stack/stack.go
  tests:
    - backend/oauth2/handlers_http/admin_key_handler_test.go
    - backend/signingkeys/keys_memory/key_store_tenant_test.go
    - backend/signingkeys/keys_vault/vault_key_store_test.go
    - backend/saml/handlers_http/scenario_examples_test.go
    - backend/shared/http/server_http/control_plane_boundary_test.go
    - backend/cmd/idmagic-batch/main_test.go
    - backend/cmd/internal/bootstrap/keystore_test.go
    - tools/check/example-coverage-debt.json
  stop_before_reading:
    - infra
    - frontend
---

# SigningKeys が宣言する具体例 14 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-496-burn-down-the-example-coverage-debt]] は具体例の被覆台帳の消化単位を Context と決め、測定のうえで残りを Context ごとの子 work item へ割った。本項目はそのうち `docs/domain/signing-keys/scenarios.feature.md` が宣言する 14 件を引き取る。

親項目が `claim-mapping` の 3 件と `workloadidentity` の 13 件で測った結果は、**16 件のうち注記だけで済んだのは 4 件だけ**だというものである。3 件はテストが 1 つも無く、9 件は既存テストへ新しい観測を足す必要があった。件数は作業量の目安にならない。

## Scope

- 14 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `// EX-SIGNINGKEYS-NNN-MM: <この具体例の何を固定しているか>` の注記を足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 注記は「何を固定しているか」を書く。id だけの注記は禁止する。
- 実装にバグがあって具体例のとおりに振る舞っていないことが分かった場合は、難しくない修正なら本 work item で修正する。難しい修正なら別 work-item に切り出す。

## Out of Scope

- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。
- `tools/check/standards-coverage-debt.json`。[[wi-495-burn-down-the-standards-coverage-debt]] が消化済みである。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。本項目が新しく書く拒否テストは、その規範を満たす形で書く。既存テストに注記を足すだけの件で、そのテストが効果の不在を見ていない場合は、注記を足したうえで wi-392 の対象として残す。
- シナリオと具体例の追加、削除、書き換え。具体例の記述が実装と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。
- ライフサイクルバッチによる `XmlFederationSigning` 鍵の周期ローテーションとアーカイブ。`EX-SIGNINGKEYS-006-01` は管理者のローテーションだけを前提にするため、[[wi-55565-rotate-xml-federation-signing-keys]] が扱う。
- SAML の専用 IdP プロファイルが持つ、デフォルト以外のスコープの鍵のローテーション。管理 API はスコープを受け取らない。
- 管理画面から `XmlFederationSigning` 鍵をローテートする操作。
- 行カバレッジ率の目標または閾値。
- `report-coverage-debt` の分類（`named`、`nearby`、`none`）を根拠にした台帳からの削除。分類は読む順を決める材料であり、台帳から外す根拠にはならない。

## 作業の進め方

[[wi-565-make-backing-declared-examples-cheap]] が、この作業の 1 件あたりの費用を下げる道具を用意した。使う。

- 読む順と観測点は `mise run spec-route -- <id>` が出す。規則、当の具体例の本文、契約が宣言する候補 operation とそのメソッド・パス・スコープ、同じ規則の隣の id を名指している既存テストが返る。**出力は読む順を決める材料であり、台帳から外す根拠にはしない。**
- 新しく書くテストは `backend/shared/http/testing_stack` の上に載せる。`Register` と同じ配線が option の合成で建ち、保存先は型付きの field から読み直せる。既存 fixture の全面移行はしない。触る必要が出た範囲だけ移す。
- 変更した Go の変異は `mise run test-go-mutation -- <package-directory>` で読む。手で書く故障注入は、変異器が表現できない「配線を外す」「既定の分岐を差し替える」種類だけに残す。
- 実装が具体例と食い違って消化できない件は、台帳の当該行へ `blocked_by`（先に決着すべき work item）と `finding`（実装が実際に何を返すか）を書いて残す。散文にだけ書くと、次の読み手が同じ測定をやり直す。

## Design

変更の中心は観測の追加である。
主要な操作は既存の管理 API のハンドラー（`handleRotateTenantKey`、`handleDisableTenantKey`、`handleListTenantKeyHealth`、`handleJWKS`）、ユースケース `ArchiveExpiredSigningKeys`、SAML の `KeyStoreSignerProvider`、バッチの `run` である。

### 食い違いの判定

| 食い違い | 具体例 | 具体例以外の根拠 | 判定 |
| --- | --- | --- | --- |
| 現在の署名鍵の無効化を、`invalid_request` ではなく `urn:idmagic:error:active_key_cannot_be_disabled` で拒否する | `EX-SIGNINGKEYS-010-02`（エラー `InvalidRequestError`） | TypeSpec の `DisableTenantKey` は 400 の本文を `InvalidRequestError`（`type` = `urn:idmagic:error:invalid_request`）だけと宣言する。この問題コードを読む画面は無い | 実装の欠陥。修正が小さいので本項目で直す |
| `XmlFederationSigning` 鍵をローテートする入口が無い | `EX-SIGNINGKEYS-006-01`（管理者が XML 鍵を K2 へローテートする） | 管理 API もバッチも用途を指定しない文脈でローテートするため、`Signing` 鍵だけが回る。鍵ストアの用途ごとのローテーションと、メタデータが猶予期間中の旧証明書を公開する振る舞いは実装済みである | 入口の欠落。利用者の判断により、本項目で契約へ入口を加える |

### XML フェデレーション鍵のローテーションの入口

利用者の判断により、既存の `RotateTenantSigningKey` に任意の本文 `AdminRotateKeyRequest { usage?: KeyUsage }` を足す。
本文が無いか `usage` が無ければ、これまでどおり `Signing` 鍵をローテートする。
`XmlFederationSigning` ではデフォルトスコープの鍵をローテートする。WS-Federation と SAML のデフォルト IdP プロファイルが署名に使う鍵である。
`usage` が `KeyUsage` のどれでもなければ、何もローテートせずに 400 `InvalidRequestError` を返す。

ハンドラーは本文を読んで用途を決め、`signingports.WithKeyUsage` を付けた文脈で既存の `usecases.RotateSigningKey` を呼ぶ。
直前の現在の鍵を読む `GetActiveKey` にも同じ文脈を渡す。
`SigningKeyRotated` は用途を持たないので、発行するイベントの形は変わらない。
猶予期間後にメタデータから旧証明書が消えるのは、`ListPublicKeys` が期限を過ぎた鍵を時刻で外すためであり、アーカイブを待たない。

### バッチの起動入力の差し替え（構造変更）

`EX-SIGNINGKEYS-003-01` と `EX-SIGNINGKEYS-012-01` の「鍵をローテートしない」「KeyStore を構築せず、署名鍵も作らない」は、設定の拒否が依存の組み立てより前に起きることで成り立つ。
これを製品と同じ `run` から観測するため、`run` が読む起動設定の ConfigLoader と依存の組み立てを引数で受ける。

```go
// launch は 1 回のバッチ起動が外界から受け取るものである。
type launch struct {
    loader   *bootstrap.ConfigLoader
    assemble func(context.Context, bootstrap.SharedConfig) (*bootstrap.Dependencies, error)
}

func run(ctx context.Context, args []string, in launch) error
```

`main` は `bootstrap.NewConfigLoader(os.Getenv)` と `bootstrap.Assemble` を渡す。起動設定を環境から読むのは ConfigLoader だけという境界の検査を保つため、`os.Getenv` そのものは渡さない。
テストは組み立ての呼び出しを数え、拒否では 0 回であることを見る。
Dependencies が組み立てられなければ KeyStore も存在しないので、鍵のローテーションも作成も起こり得ない。
振る舞いは変えないので、観測を足す前の別のコミットにする。

### テストの置き場所

| 具体例 | 置き場所 | 観測 |
| --- | --- | --- |
| `EX-SIGNINGKEYS-001-01`、`-004-01`、`-010-01`、`-010-02`、`-011-02`、`-002-01` | `backend/signingkeys/handlers_http/scenario_examples_test.go`（新設、`testing_stack`） | 管理者のログインセッションでローテーションと無効化を呼び、`/jwks` の `kid`、鍵ストアの現在の鍵、発行イベントを読み直す。API アクセストークンの拒否は `WWW-Authenticate` と問題の種類に加え、現在の鍵と JWKS が変わらないことを見る。アーカイブはバッチが呼ぶのと同じユースケースを通し、JWKS とイベントの 4 項目を読む |
| `EX-SIGNINGKEYS-008-01`、`-009-01` | 同上（`Register` を直接組む） | 到達不能にできる Vault Transit の代役と、`FindAll` の呼び出しを数えるテナント保存先を配線する。健全性の一覧、JWKS、拒否の本文、横断収集が走らないことを見る |
| `EX-SIGNINGKEYS-011-01` | `backend/oauth2/handlers_http/admin_key_handler_test.go`（既存） | 既存の観測（現在の鍵とイベント）に `access_denied` の問題の種類を足す |
| `EX-SIGNINGKEYS-005-01` | `backend/saml/handlers_http/signing_key_examples_test.go`（新設） | SSO が発行した Assertion の署名を、発行テナントの XML 証明書、他テナントの XML 証明書、発行テナントの JWT 署名鍵の 3 つで検証する |
| `EX-SIGNINGKEYS-006-01` | 同上 | 管理者のログインセッションで `usage` = `XmlFederationSigning` のローテーションを呼ぶ。次の SSO の Assertion が K2 で署名され K1 では検証できないこと、猶予期間中のメタデータに K1 と K2 があること、猶予期間後は K1 が消えることを見る。`Signing` の鍵が変わらないことも見る |
| `EX-SIGNINGKEYS-007-01` | `backend/signingkeys/keys_vault/vault_key_store_test.go`（既存） | 再起動を模した 2 つの鍵ストアで、メタデータが公開する証明書のフィンガープリントを比べる |
| `EX-SIGNINGKEYS-003-01`、`-012-01` | `backend/cmd/idmagic-batch/main_test.go`（既存） | 設定の拒否と、依存の組み立てが呼ばれないこと。対照として正しい設定では組み立てが 1 回呼ばれる |

### 選んだ実行レシピ

| 触った層 | 実行するタスク |
| --- | --- |
| Go の 1 テスト | `mise run test-go-test -- <package> <test>` |
| Go のパッケージ | `mise run test-go-package -- <package>` |
| 台帳と仕様 | `mise run check-spec` |
| frontmatter | `mise run check-work-items` |

## Tasks

- [x] T001 [Acceptance] 14 件を台帳から外し、`mise run check-spec` が 14 件を名指しで落とすことを確認する。
- [x] T001a [Spec] `RotateTenantSigningKey` に任意の本文 `AdminRotateKeyRequest` と 400 `InvalidRequestError` を宣言し、`check-api-compat` を通す。
- [x] T002 [Structure] バッチの `run` が起動設定の ConfigLoader と依存の組み立てを引数で受ける。既存テストが通ることを確認する。
- [x] T003 [Batch] `EX-SIGNINGKEYS-003-01`、`-012-01` の観測を書く。
- [x] T004 [Fix] 現在の署名鍵の無効化を `invalid_request` で拒否する（`EX-SIGNINGKEYS-010-02`）。Unit RED を先に観測する。
- [x] T005 [AdminAPI] `EX-SIGNINGKEYS-001-01`、`-002-01`、`-004-01`、`-010-01`、`-011-01`、`-011-02` の観測を書く。
- [x] T006 [Health] `EX-SIGNINGKEYS-008-01`、`-009-01` の観測を書く。
- [x] T007 [XML] `EX-SIGNINGKEYS-005-01`、`-007-01` の観測を書く。
- [x] T007a [Feature] 管理 API が `usage` に従って `XmlFederationSigning` 鍵をローテートする（`EX-SIGNINGKEYS-006-01`）。Unit RED を先に観測する。
- [x] T008 [Verify] 変異と故障注入、対象パッケージ、総合検証を通して完了記録を作る。

## Verification

- `mise run check-spec` が、`docs/domain/signing-keys/scenarios.feature.md` の 14 件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを、消化ごとに観測する。
- 予定する Acceptance RED は、台帳から 14 件を外した `mise run check-spec` である。
- 予定する Unit RED は、現在の署名鍵の無効化が `invalid_request` を返すことの `backend/signingkeys/handlers_http` の観測である。修正の無い具体例では、新しく書く各テストが観測する分岐を手で外し、そのテストが落ちることを確かめる。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。減った件数は何も意味しない。注記に「この具体例の何を固定しているか」を書かせることで区別する。親項目の測定では、16 件のうち注記だけで済んだのは 4 件だけだった。
- **`nearby` に分類された件を、テストがある証拠として読む。** 分類が言っているのは「近傍に他の拒否テストがある」だけである。親項目の測定では、`claim-mapping` の 3 件はすべて `nearby` でありながら 2 件はテストが無かった。分類は読む順の材料にとどめる。
- **拒否である具体例のテストが、応答の字面だけを見て書かれる。** 新しく書く拒否テストには [[wi-392-refusal-tests-assert-the-absent-effect]] の規範が効く。注記へ「効果の不在を何で観測したか」を書き、後から区別できるようにする。
- **具体例の `Then` が複数あるのに、観測が 1 つで済まされる。** 親項目の測定では、`EX-WORKLOADIDENTITY-008-01` のようにイベントと実際の効果の双方を言う具体例が複数あった。`Then` の数だけ観測が要る。

## Completion

- **Completed At**: 2026-09-23
- **Summary**:
  `mise run spec-diff` は `main` に対して、TypeSpec の `AdminRotateKeyRequest` の追加を返した。
  SigningKeys が台帳に残していた 14 件の具体例をすべて実装と照合し、`//spec:covers` を付けたテストへ結び付けて台帳から外した。
  管理 API と JWKS の 6 件（`001-01`、`002-01`、`004-01`、`010-01`、`010-02`、`011-02`）は `backend/signingkeys/handlers_http/scenario_examples_test.go` を新設し、`testing_stack` の上で観測した。ローテーションと無効化の効果は `/jwks` と鍵ストアの現在の鍵から読み直す。
  健全性の 2 件（`008-01`、`009-01`）は、到達不能にできる Vault Transit の代役と、全テナントの読み出しを数えるテナント保存先を配線した `health_examples_test.go` で観測した。`009-01` は Group 由来の `system_admin` を持つ他テナントの利用者を含む。
  XML フェデレーションの 2 件（`005-01`、`006-01`）は、SSO が発行した Assertion をどの証明書で検証できるかと、SAML と WS-Federation のメタデータで観測した。`007-01` は Vault の鍵ストアを作り直す前後で、メタデータが公開する証明書のフィンガープリントを比べる。`011-01` は既存テストへ `access_denied` の観測を足した。
  `003-01` と `012-01` は、バッチの `run` が起動設定の ConfigLoader と依存の組み立てを引数で受ける構造変更のうえで、設定の拒否が依存の組み立てより前に起きることを観測した。
  照合で分かった食い違いは 2 件で、どちらも本項目で直した。現在の署名鍵の無効化が、契約の `InvalidRequestError` ではなく `active_key_cannot_be_disabled` を返していた。また `XmlFederationSigning` 鍵をローテートする入口が管理 API にもバッチにも無く、`006-01` の前提に届かなかった。利用者の判断により、`RotateTenantSigningKey` へ任意の本文 `{"usage": ...}` と 400 `InvalidRequestError` を宣言し、ハンドラーが指定の用途の鍵をローテートするようにした。
  ライフサイクルバッチによる XML フェデレーション鍵の周期ローテーションとアーカイブは [[wi-55565-rotate-xml-federation-signing-keys]] へ切り出した。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`
  - **Requirement**: REQ-SIGNINGKEYS-006
  - **Observed Failure**: 台帳から 14 件を外すと、`check-spec` は `EX-SIGNINGKEYS-001-01` から `EX-SIGNINGKEYS-012-01` までのちょうど 14 件を「declared, but no test names it」で名指しして落ちた。管理 API から `usage` の配線を外すと、`TestAdministratorRotatesTheXmlFederationKeyWithoutBreakingTrust` が「ローテーション後も XML フェデレーション鍵が K1 のままである」で落ちた。
  - **Detection Reason**: 台帳の行を消すだけでは検査を通らず、各 id を名指しするディレクティブとその id を動かすテストの両方が要る。`006-01` のテストは管理 API からローテートし、SSO とメタデータで結果を読むので、入口の配線が欠けた実装と区別できる。
- **Unit RED Evidence**:
  - **Test**: `TestAdministratorCannotDisableTheCurrentSigningKey`、`TestRotationWithXmlFederationUsageRotatesOnlyTheXmlKey`、`TestRotationRefusesAnUnknownUsage`（`backend/signingkeys/handlers_http`）
  - **Requirement**: REQ-SIGNINGKEYS-006
  - **Observed Failure**: 修正前は、現在の署名鍵の無効化が 400 `urn:idmagic:error:active_key_cannot_be_disabled` を返した。`usage` = `XmlFederationSigning` のローテーションは JWT の鍵を回して `previous` が JWT の kid を指した。`usage` = `Encryption` のローテーションは 200 で JWT の鍵を回した。
  - **Detection Reason**: 3 つとも応答に加えて鍵ストアの現在の鍵を用途ごとに読み直すので、拒否を返しながらローテートする実装や、別の用途の鍵を回す実装と区別できる。
- **Change-Resistance Results**:
  `mise run test-go-mutation` は `backend/signingkeys/handlers_http` で 26 件のうち 22 件を検出した。変更した `rotationUsage`、ローテーション、無効化の拒否の変異はすべて検出された。生存した 4 件は、今回触れていない `handleListAdminKeys` と `requireKeyReader` にあり、それらを観測するテストは別パッケージ（`backend/oauth2/handlers_http`）にある。`backend/cmd/idmagic-batch` は 16 件のうち 14 件を検出した。生存した 2 件は `cadence-days <= 0` と `grace-days < 0` の境界で、`003-01` が言う `grace-days >= cadence-days` の比較の変異は検出された。
  変異器が表現できない故障を手で注入し、すべて検出された。管理 API で `usage` の配線を外す形、バッチで設定エラーを無視して組み立てへ進む形、起動設定の検査を組み立ての後へ回す形、健全性の到達性を常に真にする形、健全性一覧の前提判定を制御面主体から鍵の閲覧者へ緩める形、アーカイブのイベントの `expiresAt` を取り違える形、XML 証明書のシリアル番号を乱数にして再起動で再現できなくする形を試した。
- **Verification Results**:
  - `mise run check-spec` - 成功
  - `mise run check-api-compat` - 成功（破壊的変更なし）
  - `mise run check-contract-drift` - 成功
  - `mise run check-status-drift` - 成功
  - `mise run lint-go` - 成功
  - `mise run verify` - 成功
  - `mise run test-ui-e2e` - 未実行。画面のコードと、画面が使う本文なしのローテーションの振る舞いは変わらず、無効化の問題コードを読む画面も無い。
