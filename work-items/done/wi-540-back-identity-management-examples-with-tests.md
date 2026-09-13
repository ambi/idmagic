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
    - docs/contexts/identity-management/scenarios.feature.md
  typespec: []
  source:
    - backend/shared/http/testing_stack/stack.go
  tests:
    - backend/idmanagement/handlers_http/refusal_effects_test.go
    - backend/idmanagement/handlers_http/account_refusal_effects_test.go
    - backend/idmanagement/handlers_http/export_refusal_effects_test.go
    - backend/idmanagement/user/usecases/user_import_test.go
    - backend/idmanagement/user/usecases/admin_users_test.go
    - backend/idmanagement/group/usecases/group_import_test.go
    - backend/idmanagement/group/usecases/admin_groups_test.go
    - backend/idmanagement/agent/usecases/admin_agents_test.go
  stop_before_reading:
    - backend/idmanagement/db_postgres
    - backend/idmanagement/user/db_postgres
    - backend/idmanagement/group/db_postgres
    - frontend
    - infra
---

# IdManagement が宣言する具体例 66 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-496-burn-down-the-example-coverage-debt]] は具体例の被覆台帳の消化単位を Context と決め、測定のうえで残りを Context ごとの子 work item へ割った。本項目はそのうち `docs/contexts/identity-management/scenarios.feature.md` が宣言する 66 件を引き取る。

親項目が `claim-mapping` の 3 件と `workloadidentity` の 13 件で測った結果は、**16 件のうち注記だけで済んだのは 4 件だけ**だというものである。3 件はテストが 1 つも無く、9 件は既存テストへ新しい観測を足す必要があった。件数は作業量の目安にならない。

**66 件のうち拒否は 7 件で、59 件は拒否以外である。** また `backend/idmanagement` は `report-security-test-gaps` が挙げる既存拒否テストの最大の所有者（15 件）でもあるため、本項目が足す注記と [[wi-392-refusal-tests-assert-the-absent-effect]] の作業が同じファイルで交差する。注記だけを足して wi-392 の対象として残す件は、その旨を注記へ書く。

## Scope

- 66 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `//spec:covers EX-IDMANAGEMENT-NNN-MM: <この具体例の何を固定しているか>` のディレクティブを足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- ディレクティブは「何を固定しているか」を書く。id だけの注記は禁止する。
- 実装が具体例のとおりに振る舞っていないことが分かった場合は、**本 work item では直さず欠陥として切り出す**。

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

### 注記だけで済んだ件は 0 件だった

親項目の測定 (16 件中 4 件) より悪い。65 件のうち **40 件は既存テストが具体例の `Then` を部分的にしか見ておらず、25 件はテストが無かった**。件数は作業量の目安にならないという親項目の観察が、この Context でも再現した。

| 段階 | 件数 |
| --- | --- |
| 既存テストへ観測を足してディレクティブを付けた | 40 |
| テストを新しく書いた | 25 |
| 欠陥として切り出した | 1 |
| **注記だけで済んだ** | **0** |

部分的にしか見ていなかった 40 件の内訳は、次の 4 つに集約される。

| 既存テストが見ていなかったもの | 例 | これが無いと通る実装 |
| --- | --- | --- |
| 発行 (イベント) | 001-01 の `UserCreated`、006-01 の 4 イベント | 状態は変えるが記録を残さない |
| 保存層からの読み直し | 010-01、016-01、024-01 | 戻り値だけを組み立てる |
| 拒否が防いだ効果の不在 | 009-02、024-02、026-08 | 拒否を書いてから操作も続ける |
| 母集団の対照 | 019-01、020-01、008-01 | 対象を取り違えても同じ応答になる |

4 つ目がこの Context に固有である。IdManagement の具体例は「自分のものだけ」「そのグループのメンバーだけ」「条件に一致する有効な User だけ」と、**集合を絞ることそのものを述べる**ものが多い。1 人しか居ないテナント、1 グループしか無いファイル、メンバーが 1 人のグループを前提にしたテストは、絞り込みを一切していない実装をそのまま通す。消化のたびに「対照になる 2 人目」を置く必要があった。

### 拒否以外の観測の型 (Authentication との差)

[[wi-539-back-authentication-examples-with-tests]] が固めた 4 つの型のうち、「〜画面へ進む」と「省略の可否と `amr`/`acr`」はこの Context に現れない。代わりに CSV の経路が持ち込んだ型が 2 つある。

| 具体例の `Then` の形 | 観測 | これが無いと通る実装 |
| --- | --- | --- |
| 「preview は `User` を変更しない」 | preview の前後で保存層の集合を写し取って比較 | preview で書いてしまう |
| 「行は不可分に保存される」 | 失敗した行の 4 つの書き込み先をすべて読み直す | 行の一部だけを書く |

後者は 004-08 と 026-11 が同じ形で述べており、どちらも既存テストは確定境界へ渡った回数しか見ていなかった。回数は「呼ばれなかった」ことしか言わず、「呼ばれる前に別経路で書いた」を落とせない。

### 基盤へ足したもの

`idmRefusalFixture` (`backend/idmanagement/handlers_http`) へ 4 つ足した。どれも「その具体例を観測するには基盤が足りない」と分かった時点で足している。

- `OAuth2.ConsentRepo`: アカウントのデータエクスポートが読む先 (002-01)。未配線のまま nil 参照で落ちていた。
- `Authentication.PasswordHistoryRepo`: 管理 API の作成が履歴の追記まで進む (014-01)。拒否だけを見ていた間は到達しなかった経路である。
- `OAuth2.ClientRepo` とテナントごとのクライアント 2 つ: 越境した資格情報のバインド (009-04)。
- `runExport` への `Deps.Emit` 配線: **これが本題だった。** エクスポートの生成は 3 段のいずれもイベントとしてしか観測できないのに、既存の `runExport` は `Emit` を配線しておらず、`DataExportStarted` も `DataExportSucceeded` も 1 度も読まれていなかった。wi-539 が Authentication の 4 fixture で見つけたのと同じ欠落である。

### `spec:covers` は 1 行に収める

`golangci-lint fmt` (gofumpt) は、ディレクティブの形をしたコメントを doc comment ブロックの末尾へ動かす。複数行にまたがるディレクティブを書くと、2 行目以降が先頭へ残って本文が分断される。既存のディレクティブがどれも文の途中で切れているのはこのためである。

**本項目が書いたディレクティブはすべて 1 行で完結させ、説明はその上の段落へ置いた。** 整形を通しても意味が保たれる形はこれだけである。

## Tasks

- [x] T001 [Acceptance] `mise run check-spec` が対象 66 件を名指しで落とすことを、消化前に観測する。
- [x] T002 REQ-IDMANAGEMENT-001 から 003 を消化する。フェデレーション JIT、account スコープ、メールアドレス確認の CSRF 境界。`mise run test-go-package -- ./backend/idmanagement/handlers_http`。
- [x] T003 REQ-IDMANAGEMENT-004 と 007 を消化する。User CSV のインポートとエクスポートの往復。`mise run test-go-package -- ./backend/idmanagement/user/usecases`。
- [x] T004 REQ-IDMANAGEMENT-005 と 006 と 008 を消化する。一覧のページングとエクスポートのジョブ。`runExport` へイベントの記録を足した。
- [x] T005 REQ-IDMANAGEMENT-009 から 011、013 から 016、018 から 021、023、024 を消化する。Agent、ライフサイクル、グループ、アカウント。
- [x] T006 REQ-IDMANAGEMENT-025 を消化する。スコープの粒度。
- [x] T007 REQ-IDMANAGEMENT-026 から 028 を消化する。Group CSV のインポート、エクスポート、行操作による削除。
- [x] T008 [Decision] `EX-IDMANAGEMENT-009-04` を [[wi-573-agent-admin-api-answers-with-undeclared-statuses]] として切り出し、台帳へ `blocked_by` と `finding` を書いた。
- [x] T009 [Change-Resistance] `group/usecases` と `user/usecases` で `mise run test-go-mutation` を読み、範囲内の生き残り 5 件を殺した。
- [x] T010 [Verify] `mise run verify`。

## Verification

- `mise run check-spec` が、`docs/contexts/identity-management/scenarios.feature.md` の 66 件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを、消化ごとに観測する。
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
  `mise run spec-diff -- 2a18c26f` は「no normative specification change」を返す。規範は動いていない。
  動いたのは対応付けで、IdManagement が宣言する具体例のうち本項目が引き取った 66 件から 65 件を
  `tools/check/example-coverage-debt.json` から外し、`//spec:covers` を持つテストへ結び付けた。
  台帳の IdManagement 行は 66 件から 1 件へ減り、残る 1 件は `blocked_by` と `finding` を持つ。
  **製品コードは 1 行も変えていない。** 変わったのはテストと、テストが使う fixture の配線である。
  併せて `idmRefusalFixture` が同意・パスワード履歴・テナントごとの OAuth2 クライアントを配るようになり、
  `runExport` がエクスポートの発行を記録するようになった。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`
  - **Requirement**: REQ-IDMANAGEMENT-004
  - **Observed Failure**: 対象 66 件を台帳から外した状態で、66 件すべてを名指しで落とした。例:
    `docs/contexts/identity-management/scenarios.feature.md:7: EX-IDMANAGEMENT-001-01 is declared, but no test names it.`
  - **Detection Reason**: 検査は「その id を名指したテストが存在するか」だけを見る。
    台帳から外したうえで落ちることを先に観測しているので、通ったことはディレクティブが実在することを意味する。
    ディレクティブの中身が空でないことは検査では読めないため、そこは各テストで `Then` の数だけ観測を置いた。
- **Unit RED Evidence**:
  - **Test**: `TestProvisionFederatedUserCreatesCredentiallessActiveUser`
    (`backend/idmanagement/user/usecases`)
  - **Requirement**: REQ-IDMANAGEMENT-001
  - **Observed Failure**: `federated_user.go` の `UserCreated` 発行を `if false` で括って配線を外すと
    `admin_users_test.go:181: UserCreated was not emitted, events=[]` で落ちる。元に戻すと通る。
  - **Detection Reason**: この経路は作成した `User` を戻り値で返すので、応答からは発行の有無が見えない。
    元のテストは `Deps.Emit` を配線しておらず、発行を 1 度も読んでいなかった。
    具体例の 2 つ目の `Then` (「`UserCreated` を発行する」) に対応する観測が無い状態だった。
- **Change-Resistance Results**:
  本項目が触れた製品コードは無い。したがって変異が測るのは「足した観測が検出力を上げたか」である。
  `mise run test-go-mutation` (gremlins) を 2 つの package で読んだ。
  `./backend/idmanagement/group/usecases` は Killed 312 / Lived 41 / Not covered 174 (efficacy 88.39%)、
  `./backend/idmanagement/user/usecases` は Killed 318 / Lived 34 / Not covered 25 (efficacy 90.34%)。
  生き残り 75 件を読み、**本項目が引き取った具体例の内側にあった 5 件を殺した。**
  - `dynamic_groups.go:89` と `:122` の `Version + 1` → `- 1`。版が戻る実装でも
    「旧版は除外される」は成立してしまう。除外は版の不一致で起きるためである。
    しかし版が戻れば、いつか過去の版と一致して消えたはずの所属を復活させる。
    023-01 と 020-01 へ「版が単調に進むこと」の観測を足した。
  - `dynamic_groups.go:193` の `len(userIDs) > 100` の境界。021-01 の `Given` そのものである。
    100 件は通り 101 件は落ちることを、境界の両側で見る形へ足した。
  - `dynamic_groups.go:238` の `evalErr != nil`。021-01 の「判定を返す」には
    「評価に失敗した User は判定ではなく `error_code` を伴う」が含まれる。観測が無かった。
  - `dynamic_groups.go:339` の `*existing.RuleVersion == rule.Version`。負にすると再評価が
    旧版の行を「有効」と読み、書き直さずに残す。所属は除外されたまま誰にも権限が戻らない。
    023-01 へ「再評価後の所属が新しい版を持ち、実効ロールが戻ること」を足した。
  残る 70 件は範囲外として残す。内訳は 3 種類である。
  1. `AdminEmit` の戻り値検査 (`err != nil` の否定)。テストの `Emit` は決して失敗しないので、
     否定しても振る舞いが変わらない。殺すには outbound provisioning の通知先を配線する必要があり、
     それは wi-45 の主題である。
  2. ページングの前進を守る防御的な判定 (`group_import_planner.go:125` など)。
     壊れた repository を注入しなければ踏めない。
  3. `changed_fields` の算出 (`user_import_apply.go:109`、`:114`)。User の具体例は
     `changed_fields` を述べていない。Group の側 (024-01) は述べており、そちらは殺してある。
  手書きの故障注入は 2 つ行った。どちらも変異器が表現できない種類である。
  1 つは上記の Unit RED (発行の配線を外す)。もう 1 つは `groups.go` の `membershipEffective` から
  `*member.RuleVersion == rule.Version` を落とすもので、
  `版が上がったのに旧版の所属が実効ロールに残った` で落ちることを観測した。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - 実行していない。本項目は frontend のファイルを 1 つも変更しておらず、
    製品コードも 1 行も動いていない。ブラウザーへ届く変更が無いため、後退の起きようがない。
