---
depends_on: [wi-565-make-backing-declared-examples-cheap]
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの具体例に、その id を名指しするテストを対応付ける作業である。シナリオも製品の振る舞いも変えない。テストが書けない具体例が見つかった場合、それは実装が具体例のとおりに振る舞っていないということなので、欠陥として個別の work item に切り出す。" }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 宣言済みの具体例に検証を与える作業であり、公開契約も運用手順も変わらない。
  references: []
initial_context:
  specification:
    - docs/domain/system/scenarios.feature.md
  typespec: []
  source:
    - tools/check/src/normative-coverage.ts
    - tools/check/src/check-documents.ts
    - backend/shared/http/server_http/health_handler.go
    - backend/shared/http/support_http/deprecation.go
    - backend/shared/http/support_http/error_handler.go
    - backend/shared/storage/db_postgres/base.go
    - frontend/src/lib/i18n
    - frontend/src/api/core.ts
    - frontend/src/api/oidc.ts
    - frontend/src/routes/index.tsx
    - frontend/src/features/auth-flow/HomePage.tsx
    - frontend/src/components/LanguageSwitcher.tsx
    - infra/k8s/base/api.yaml
    - infra/k8s/monitoring
    - infra/docker/prometheus.yml
    - infra/docker/prometheus-rules.yml
  tests:
    - backend/shared/http/server_http/health_handler_test.go
    - backend/shared/http/server_http/routes_e2e_test.go
    - backend/shared/http/server_http/priority_class_test.go
    - backend/shared/http/support_http/deprecation_test.go
    - backend/shared/http/support_http/admission_test.go
    - backend/shared/storage/db_postgres/base_test.go
    - backend/shared/observability/metrics_prometheus/metrics_test.go
    - backend/cmd/internal/bootstrap
    - frontend/src/lib/i18n/resolveLocale.test.ts
    - frontend/src/lib/i18n/errorMessage.test.ts
    - frontend/src/api/core.test.ts
    - frontend/src/api/oidc.test.ts
    - frontend/src/components/LanguageSwitcher.test.tsx
    - frontend/tests/e2e/localized-ui.spec.ts
    - frontend/tests/e2e/admin-session-recovery.spec.ts
  stop_before_reading:
    - docs/requirements/quality.md
    - docs/design/observability/monitoring.md
    - frontend/src/features/admin-users
    - infra/k8s/monitoring/loki
---

# System が宣言する具体例 45 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-496-burn-down-the-example-coverage-debt]] は具体例の被覆台帳の消化単位を Context と決め、測定のうえで残りを Context ごとの子 work item へ割った。本項目はそのうち `docs/domain/system/scenarios.feature.md` が宣言する 45 件を引き取る。

親項目が `claim-mapping` の 3 件と `workloadidentity` の 13 件で測った結果は、**16 件のうち注記だけで済んだのは 4 件だけ**だというものである。3 件はテストが 1 つも無く、9 件は既存テストへ新しい観測を足す必要があった。件数は作業量の目安にならない。

**`system` には対応する Go パッケージが無い。** `mise run report-coverage-debt` は要求 id の接頭辞と `backend/` 直下のディレクトリ名を突き合わせて候補を出すため、45 件すべてが候補 0 件の `none` に落ちている。これは「テストが無い」ことの証拠ではなく、報告が見ていないことの証拠である。**本項目は、まずどのパッケージのテストが所有するかを決めることから始める。**

## Scope

- 45 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `//spec:covers EX-SYSTEM-NNN-MM: <この具体例の何を固定しているか>` のディレクティブを足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- ディレクティブは「何を固定しているか」を書く。id だけの注記は禁止する。`//spec:covers` の形だけが被覆と数えられる。
- 実装が具体例のとおりに振る舞っていないことが分かった場合は、**本 work item では直さず欠陥として切り出す**。
- 45 件の所有パッケージを決めて記録する。決めた対応を `report-coverage-debt` の `packageFor` へ反映するかどうかも、消化の過程で判断する。

## Out of Scope

- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。
- `tools/check/standards-coverage-debt.json`。[[wi-495-burn-down-the-standards-coverage-debt]] が消化済みである。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。本項目が新しく書く拒否テストは、その規範を満たす形で書く。既存テストに注記を足すだけの件で、そのテストが効果の不在を見ていない場合は、注記を足したうえで wi-392 の対象として残す。
- シナリオと具体例の追加、削除、書き換え。具体例の記述が実装と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。
- 見つかった実装の欠陥の修正。切り出した先で扱う。
- 行カバレッジ率の目標または閾値。
- `report-coverage-debt` の分類（`named`、`nearby`、`none`）を根拠にした台帳からの削除。分類は読む順を決める材料であり、台帳から外す根拠にはならない。

## 作業の進め方

[[wi-565-make-backing-declared-examples-cheap]] が、この作業の 1 件あたりの費用を下げる道具を用意した。使う。

- 読む順と観測点は `mise run spec-route -- <id>` が出す。規則、当の具体例の本文、契約が宣言する候補 operation とそのメソッド・パス・スコープ、同じ規則の隣の id を名指している既存テストが返る。**出力は読む順を決める材料であり、台帳から外す根拠にはしない。**
- 新しく書くテストは `backend/shared/http/testing_stack` の上に載せる。`Register` と同じ配線が option の合成で建ち、保存先は型付きの field から読み直せる。既存 fixture の全面移行はしない。触る必要が出た範囲だけ移す。
- 変更した Go の変異は `mise run test-go-mutation -- <package-directory>` で読む。手で書く故障注入は、変異器が表現できない「配線を外す」「既定の分岐を差し替える」種類だけに残す。
- 実装が具体例と食い違って消化できない件は、台帳の当該行へ `blocked_by`（先に決着すべき work item）と `finding`（実装が実際に何を返すか）を書いて残す。散文にだけ書くと、次の読み手が同じ測定をやり直す。

## Design

### 45 件の所有パッケージ

`system` に対応する Go パッケージが無いというのは、正確には「1 つではない」である。この Context は**プロセスの入口**を持つので、宣言する振る舞いは frontend、`backend/shared`、`backend/cmd`、そして運用資材 (`infra/`) に分かれて置かれている。読む順を決めるために、規則ごとに所有を先に決めた。

| 規則 | 所有 | 観測の性質 |
| --- | --- | --- |
| REQ-SYSTEM-001 | `backend/shared/http/server_http`、`backend/shared/observability/metrics_prometheus` | 運用資材とエンドポイントの対応 |
| REQ-SYSTEM-002 | `backend/shared/http/server_http` | プローブの応答 |
| REQ-SYSTEM-003、004、005、008 | `frontend/src/lib/i18n`、`frontend/src/components` | 表示言語の解決 |
| REQ-SYSTEM-006、007 | `frontend/src/features/auth-flow`、`frontend/src/routes` | 起動時設定による導線の有無 |
| REQ-SYSTEM-009、010 | `frontend/src/components`、`frontend/tests/e2e` | 画面全体への反映 |
| REQ-SYSTEM-011 | `frontend/src/lib/i18n`、`frontend/src/api` | エラーコードの翻訳 |
| REQ-SYSTEM-012 | `backend/shared/storage/db_postgres` | 期限とサーキットブレーカー |
| REQ-SYSTEM-013 | `backend/shared/http/support_http` | エラー本文の言語 |
| REQ-SYSTEM-014 | `backend/shared/http/support_http` | 非推奨ヘッダー |
| REQ-SYSTEM-015 | `frontend/src/api`、`frontend/tests/e2e` | 失効セッションからの復帰 |
| REQ-SYSTEM-016、017 | `backend/cmd/internal/bootstrap`、`backend/shared/http/server_http` | 起動時設定の検証と生成物 |
| REQ-SYSTEM-018、019 | `backend/shared/http/support_http`、`backend/shared/http/server_http`、`backend/cmd/idmagic` | 入場制御と分類の生成物 |

**`report-coverage-debt` の `packageFor` は変えない。** `packageFor` は id の接頭辞から `backend/` 直下のディレクトリ名を 1 つ返す関数であり、上の表のように 1 つの規則が `backend/shared/*`、`backend/cmd/*`、`frontend/` に分かれる対応は、戻り値の形そのものが表現できない。無理に 1 つ選べば、残りを見ていないことを隠したうえで「候補あり」と報告することになり、いまの候補 0 件よりも悪い。報告が `system` を扱えないことは事実として残し、所有はこの表が持つ。

### 運用資材を名指す具体例は、資材とエンドポイントの対応として観測する

EX-SYSTEM-001-01 と 001-03 の `Then` は `infra/` の資材について述べる。被覆の判定に使われるテスト集合は `backend/` と `frontend/` の下のテストファイルだけなので (`check-documents.ts` の `PRODUCT_TREES`)、これらを消化するには製品側のテストが資材を読む必要がある。`backend/shared/storage/db_postgres/schema_test.go` が `infra/schema/postgres.sql` を読む先例があり、同じ形を採る。

その形を採ることには独立した価値がある。kubeconform と kustomize はスキーマと構文しか見ないので、プローブのパスを `/healthz` に書き換えても、`ServiceMonitor` の `path` を `/metric` にしても、いまは何も落ちない。資材と、`Register` が実際に登録する経路を突き合わせるテストだけが、その書き換えを落とす。

### 具体例の `Then` を数える

**注記だけで済んだのは 45 件のうち 3 件だけだった。** 親項目の測定 (16 件中 4 件) と同じ比率である。

| 段階 | 件数 |
| --- | --- |
| 既存テストへ観測を足してディレクティブを付けた | 22 |
| テストを新しく書いた | 18 |
| 注記だけで済んだ | 3 |
| 欠陥として切り出した | 2 |

注記だけで済んだのは `EX-SYSTEM-014-01`、`014-02`、`014-03` である。非推奨ヘッダーの 3 つの具体例が、1 つの表駆動テストの 3 行にそのまま対応していた。

既存テストが見ていなかったものは 4 つの型に集約される。

| 既存テストが見ていなかったもの | 例 | これが無いと通る実装 |
| --- | --- | --- |
| 区別の相手側 | 002-03 の「生存確認は healthy を維持する」 | 依存障害で liveness も落とす |
| 本文 (状態コードだけを見ている) | 002-01 の `200 healthy` | 空の 200 を返す |
| 拒否が防いだ効果 | 016-02 から 016-04 の「副作用のある初期化を開始せず終了する」 | 検証より先に listener を開き PostgreSQL へつなぐ |
| 資源の解放 | 012-01、012-02 の「接続が解放される」 | cancel を呼ばず 1 要求ごとに接続を漏らす |

3 つめが本項目でいちばん大きい。設定の読み込み関数だけを呼ぶテストは「集約されたエラーが返る」までしか言えず、そのエラーが返る前に副作用を起こす実装を落とせない。`Run()` の入口から当てる新しいテストで、`DATABASE_URL` を閉じたポートへ向け、返るエラーが設定の読み込みであることを観測した。検証の門を外すと、同じテストが接続の失敗 (7.5 秒) を報告して落ちる。

### 起動時設定は入力として渡せる形にする

`routes/index.tsx` の loader が `import.meta.env.DEV || import.meta.env.VITE_DEMO_LOGIN_ENABLED === 'true'` を式として持っていたため、起動時設定を入力として与える方法が無く、006-02 と 007-01 のどちらも観測できなかった。`configuredDefaultLocale(value = import.meta.env.VITE_DEFAULT_LOCALE)` と同じ形へ寄せ、`demoLoginEnabled(env = import.meta.env)` の既定引数を境界にした。振る舞いは変わらない。

`RenderConfigReference()` にも同じ形を当てた。乖離を報告する経路 (説明の無いキー、どのプロセスも読まないキー) は、製品の表を書き換えないと踏めず、書き換えると並行して走る他のテストと競合する。節・説明・registry を引数で受ける `renderConfigReference` を内側に置き、公開関数は製品の表を渡すだけにした。

### 消化できなかった 2 件

どちらも具体例と実装が食い違っており、テストを書けばどちらか一方を固定してしまう。台帳へ `blocked_by` と `finding` を残した。

- `EX-SYSTEM-008-02` → [[wi-575-explicit-display-language-outranks-the-ui-locales-hint]]。解決順は `ui_locales` ヒント > 保存済み設定であり、明示選択は保存済み設定として読み直される。認可リクエストはページの寿命を新しくするので、`ja` を明示選択済みの利用者へ `ui_locales=en` が来ると `en` が出る。既存テストがこの順序を明示的に固定している。
- `EX-SYSTEM-010-03` → [[wi-576-missing-translation-key-has-no-runtime-fallback]]。キー単位の実行時フォールバックは存在せず、`defineDictionary` の型検査が欠落そのものを防いでいる。別の機構を具体例の観測として数えないために残した。

## Tasks

- [x] T001 [Acceptance] `mise run check-spec` が対象 45 件を名指しで落とすことを、消化前に観測する。
- [x] T002 REQ-SYSTEM-001 と 002 を消化する。運用資材とプローブ。`ops_assets_test.go` を新設し、プローブとスクレイプの対応を `Register` の登録経路と突き合わせた。`mise run test-go-package -- ./backend/shared/http/server_http`。
- [x] T003 REQ-SYSTEM-012 を消化する。クエリの期限と接続の解放。`mise run test-go-package -- ./backend/shared/storage/db_postgres`。
- [x] T004 REQ-SYSTEM-013 と 014 を消化する。エラー本文の言語と非推奨ヘッダー。`mise run test-go-package -- ./backend/shared/http/support_http`、`./backend/oauth2/handlers_http`。
- [x] T005 REQ-SYSTEM-016 と 017 を消化する。起動時設定の検証と生成物。`Run()` からの拒否と、生成物の乖離報告を新設した。
- [x] T006 REQ-SYSTEM-018 と 019 を消化する。入場制御と分類の生成物。
- [x] T007 REQ-SYSTEM-003、004、005、008 を消化する。表示言語の解決。`mise run test-ui-unit-file -- src/lib/i18n/resolveLocale.test.ts`。
- [x] T008 REQ-SYSTEM-006 と 007 を消化する。起動時設定による DemoLoginAffordance と、資格情報を持つプロファイル。
- [x] T009 REQ-SYSTEM-009、010、011 を消化する。画面全体への反映とエラーコードの翻訳。
- [x] T010 REQ-SYSTEM-015 を消化する。失効セッションからの復帰。
- [x] T011 [Decision] `EX-SYSTEM-008-02` と `EX-SYSTEM-010-03` を切り出し、台帳へ `blocked_by` と `finding` を書いた。
- [x] T012 [Change-Resistance] 手書きの故障注入 8 件で、足した観測が検出力を持つことを確かめた。変異器による読み取りは [[wi-574-replace-gremlins-with-gomutants]] が Gremlins を差し替える最中なので、この記録では回していない。
- [x] T013 [Verify] `mise run verify`。frontend に触れるため `mise run test-ui-e2e` も回した。

## Verification

- `mise run check-spec` が、`docs/domain/system/scenarios.feature.md` の 45 件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを、消化ごとに観測する。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。減った件数は何も意味しない。注記に「この具体例の何を固定しているか」を書かせることで区別する。親項目の測定では、16 件のうち注記だけで済んだのは 4 件だけだった。
- **`nearby` に分類された件を、テストがある証拠として読む。** 分類が言っているのは「近傍に他の拒否テストがある」だけである。親項目の測定では、`claim-mapping` の 3 件はすべて `nearby` でありながら 2 件はテストが無かった。分類は読む順の材料にとどめる。
- **拒否である具体例のテストが、応答の字面だけを見て書かれる。** 新しく書く拒否テストには [[wi-392-refusal-tests-assert-the-absent-effect]] の規範が効く。注記へ「効果の不在を何で観測したか」を書き、後から区別できるようにする。
- **具体例の `Then` が複数あるのに、観測が 1 つで済まされる。** 親項目の測定では、`EX-WORKLOADIDENTITY-008-01` のようにイベントと実際の効果の双方を言う具体例が複数あった。`Then` の数だけ観測が要る。

## Completion

- **Completed At**: 2026-09-13
- **Summary**:
  `mise run spec-diff` は「no normative specification change against main」を返す。規範は動いていない。
  動いたのは対応付けで、System が宣言する具体例 45 件のうち **43 件**を
  `tools/check/example-coverage-debt.json` から外し、`//spec:covers` を持つテストへ結び付けた。
  台帳の System 行は 45 件から 2 件へ減り、残る 2 件はどちらも `blocked_by` と `finding` を持つ。
  消化の内訳は、既存テストへ観測を足したもの 22 件、新しく書いたもの 18 件、注記だけで済んだもの 3 件である。
  **製品コードの振る舞いは変えていない。** 変えたのは 2 つの境界だけで、どちらも入力の受け取り方である。
  `routes/index.tsx` の loader が式で持っていた起動時設定の判定を `demoLoginEnabled(env = import.meta.env)` へ
  出し、`RenderConfigReference()` の内側に節・説明・registry を引数で受ける `renderConfigReference` を置いた。
  新しく建てた観測点は 5 つのファイルである。運用資材と登録済み経路の突き合わせ
  (`ops_assets_test.go`)、起動時設定の拒否が副作用の前で終わること
  (`startup_refusal_e2e_test.go`)、`development` プロファイルだけがデモ資格情報を持つこと
  (`demo_login_seed_test.go`)、API のエラー本文が英語であること
  (`error_language_examples_test.go`)、生成物の乖離が再生成を促すこと
  (`idmagic-route-reference/main_test.go`) である。
  基盤へは `testing_stack` の `Browser.PostRawJSON` を足した。`json.Marshal` を通す入口では、
  復号が失敗する要求そのものを作れなかった。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`
  - **Requirement**: REQ-SYSTEM-016
  - **Observed Failure**: 対象 45 件を台帳から外した状態で、45 件すべてを名指しで落とした。例:
    `docs/domain/system/scenarios.feature.md:7: EX-SYSTEM-001-01 is declared, but no test names it.`
  - **Detection Reason**: 検査は「その id を名指したテストが存在するか」だけを見る。
    台帳から外したうえで落ちることを先に観測しているので、通ったことはディレクティブが実在することを意味する。
    ディレクティブの中身が空でないことは検査では読めないため、そこは各テストで `Then` の数だけ観測を置き、
    消化ごとに故障注入で確かめた。
- **Unit RED Evidence**:
  - **Test**: `TestRunRefusesInvalidStartupConfigurationBeforeAnySideEffect`
    (`backend/cmd/idmagic/startup_refusal_e2e_test.go`)
  - **Requirement**: REQ-SYSTEM-016
  - **Observed Failure**: `server.go` から `if err := loader.Err(); err != nil { ... }` の 3 行を外すと
    `Run() error = assemble dependencies: failed to connect to ...: dial error ...,
    want the startup configuration to be refused before assembly` で落ちた。所要 7.5 秒。
    元に戻すと 0.6 秒で通る。
  - **Detection Reason**: 設定を読む関数だけを呼ぶテストは「集約されたエラーが返る」までしか言えず、
    そのエラーが返る前に listener を開き PostgreSQL へつなぐ実装を落とせない。
    `DATABASE_URL` を閉じたポートへ向けてあるので、検証より先に Assemble へ進めばエラーの種類が変わり、
    落ちた時間そのものが接続を試みた証拠になる。
- **Change-Resistance Results**:
  変異器による読み取りは行っていない。[[wi-574-replace-gremlins-with-gomutants]] が Gremlins を
  差し替える最中であり、置き換わる道具の出力を証拠として残す意味が無いためである。
  本項目は製品の振る舞いを変えていないので、測るべきは「足した観測が検出力を上げたか」である。
  それを手書きの故障注入 8 件で確かめた。変異器が表現できない「配線を外す」「既定の分岐を差し替える」
  種類を選んでいる。
  1. `infra/k8s/base/api.yaml` の `livenessProbe` を `/healthz` へ。
     `TestOperationalManifestProbesCallTheRegisteredEndpoints` が両方向で落ちた
     (資材が登録されていないパスを指すこと、登録済みの `/livez` を誰も呼ばないこと)。
  2. `handleLivez` へ `DbPing` の判定を足す。`TestReadinessReportsDependencyFailure` が
     `liveness status=503` で落ちた。依存障害で liveness まで落とす実装を区別できている。
  3. `handleLivez` の `status` を `healthy` から `alive` へ。同テストと `TestHealthProbes` が
     3 つの段階すべてで落ちた。状態コードだけを見ていた元のテストはこれを通していた。
  4. `resilientRow.Scan` の先頭で `cancel()` を呼ぶ。
     `TestResilientDBQueryRowDoesNotCancelBeforeScan` が `error = context canceled` で落ちた。
  5. 同じ `Scan` から `r.cancel()` を落とす。同テストが
     `query context error after Scan = <nil>, want canceled` で落ちた。期限の保持と解放の両方向を
     区別できている。
  6. `authorize_login.go` の不正 JSON の `detail` を日本語へ、`authorize_completion.go` の
     `consent_required` の説明を空へ。`TestAPIErrorTextStaysEnglish` が 2 件同時に落ちた。
  7. `server.go` から設定検証の門を外す (上の Unit RED)。
  8. `demoLoginEnabled` の `=== 'true'` を `!== undefined` へ。`HomePage.test.tsx` の
     起動時設定の事例が落ちた。値が何であれ導線を出す実装を区別できている。
  範囲外として残したものが 1 つある。`EX-SYSTEM-012-01` と `012-02` に足した
  「取得済み接続数が基準へ戻る」の観測は、この環境では embedded-postgres が
  `shmget: No space left on device` で起動できず skip する。同じ具体例に対しては
  偽物のプールを使う `contextCheckingDB` の側で期限の保持と解放を観測してあり、
  上の 4 と 5 はそちらへの注入である。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - passed (28 tests)
