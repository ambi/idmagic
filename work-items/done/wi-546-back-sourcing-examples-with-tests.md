---
depends_on: [wi-565-make-backing-declared-examples-cheap]
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの具体例に、その id を名指しするテストを対応付ける作業である。シナリオは変えない。本項目で直した欠陥は、実装を宣言済みの具体例へ合わせるものであり、規範を変えない。" }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 宣言済みの具体例と既存の検証を対応付ける保守作業であり、公開契約と運用手順を変えない。
  references: []
initial_context:
  specification:
    - docs/domain/sourcing/scenarios.feature.md#REQ-SOURCING-001
    - docs/domain/sourcing/scenarios.feature.md#REQ-SOURCING-002
    - docs/domain/sourcing/scenarios.feature.md#REQ-SOURCING-003
    - docs/domain/sourcing/scenarios.feature.md#REQ-SOURCING-004
    - docs/domain/sourcing/scenarios.feature.md#REQ-SOURCING-005
    - docs/domain/sourcing/scenarios.feature.md#REQ-SOURCING-006
    - docs/domain/sourcing/scenarios.feature.md#REQ-SOURCING-007
  typespec: []
  source:
    - backend/sourcing/scim/domain/mutation.go
    - backend/sourcing/scim/usecases/users.go
    - backend/apitoken/usecases/usecases.go
    - backend/shared/http/support_http/auth.go
  tests:
    - backend/sourcing/scim/handlers_http/scim_test.go
    - backend/sourcing/scim/handlers_http/resource_contract_test.go
    - backend/sourcing/scim/handlers_http/standards_test.go
    - backend/sourcing/scim/domain/mutation_test.go
    - backend/sourcing/scim/usecases/users_test.go
    - tools/check/example-coverage-debt.json
  stop_before_reading:
    - frontend
    - infra
    - backend/sourcing/scim/db_postgres
    - backend/provisioning
---

# Sourcing が宣言する具体例 25 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-496-burn-down-the-example-coverage-debt]] は具体例の被覆台帳の消化単位を Context と決め、測定のうえで残りを Context ごとの子 work item へ割った。本項目はそのうち `docs/domain/sourcing/scenarios.feature.md` が宣言する 25 件を引き取る。

親項目が `claim-mapping` の 3 件と `workloadidentity` の 13 件で測った結果は、**16 件のうち注記だけで済んだのは 4 件だけ**だというものである。3 件はテストが 1 つも無く、9 件は既存テストへ新しい観測を足す必要があった。件数は作業量の目安にならない。

## Scope

- 25 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `// EX-SOURCING-NNN-MM: <この具体例の何を固定しているか>` の注記を足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 注記は「何を固定しているか」を書く。id だけの注記は禁止する。
- 実装が具体例のとおりに振る舞っていないことが分かった場合は、**本 work item では直さず欠陥として切り出す**。ただし、利用者が本項目での修正を選んだ小さな欠陥は、ここで直す。

## Out of Scope

- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。
- `tools/check/standards-coverage-debt.json`。[[wi-495-burn-down-the-standards-coverage-debt]] が消化済みである。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。本項目が新しく書く拒否テストは、その規範を満たす形で書く。既存テストに注記を足すだけの件で、そのテストが効果の不在を見ていない場合は、注記を足したうえで wi-392 の対象として残す。
- シナリオと具体例の追加、削除、書き換え。具体例の記述が実装と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。
- 見つかった実装の欠陥の修正。切り出した先で扱う。例外は「実装の欠陥の修正」節の 1 件である。
- 行カバレッジ率の目標または閾値。
- `report-coverage-debt` の分類（`named`、`nearby`、`none`）を根拠にした台帳からの削除。分類は読む順を決める材料であり、台帳から外す根拠にはならない。

## 作業の進め方

[[wi-565-make-backing-declared-examples-cheap]] が、この作業の 1 件あたりの費用を下げる道具を用意した。使う。

- 読む順と観測点は `mise run spec-route -- <id>` が出す。規則、当の具体例の本文、契約が宣言する候補 operation とそのメソッド・パス・スコープ、同じ規則の隣の id を名指している既存テストが返る。**出力は読む順を決める材料であり、台帳から外す根拠にはしない。**
- 新しく書くテストは `backend/shared/http/testing_stack` の上に載せる。`Register` と同じ配線が option の合成で建ち、保存先は型付きの field から読み直せる。既存 fixture の全面移行はしない。触る必要が出た範囲だけ移す。
- 変更した Go の変異は `mise run test-go-mutation -- <package-directory>` で読む。手で書く故障注入は、変異器が表現できない「配線を外す」「既定の分岐を差し替える」種類だけに残す。
- 実装が具体例と食い違って消化できない件は、台帳の当該行へ `blocked_by`（先に決着すべき work item）と `finding`（実装が実際に何を返すか）を書いて残す。散文にだけ書くと、次の読み手が同じ測定をやり直す。

## Design

### 証拠の境界

本項目は規範を変えず、宣言済みの具体例とテストの対応を検証可能な形にする。
製品ロジックの変更は「実装の欠陥の修正」節の 1 件だけである。

Acceptance RED は、台帳から対象 25 件を外した時点の `mise run check-spec` とする。
検査が具体例 id を名指しで落とすため、対応する `//spec:covers` がなければ完了できない。
修正した欠陥については、`EX-SOURCING-007-03` の HTTP テストの PUT と PATCH の事例を Acceptance RED とする。

対応付けだけの 24 件では Unit RED を N/A とし、代替の RED を同じ `mise run check-spec` とする。
修正した欠陥の Unit RED は `TestWriteUserRefusedByAnUnresolvableManagerLeavesTheStoredUserUnchanged` とする。

### 実装の欠陥の修正

`EX-SOURCING-007-03` の照合で、`PERSISTENCE=memory` の構成では拒否した書き込みが User を変えることが分かった。
テナントに存在しない manager を指す UpdateScimUser は 400 `invalidValue` を返しながら、本文が省略したメールアドレスを消す。
先に別の操作を並べた PatchScimUser も、manager の解決に失敗するまでの操作（`active`、`emails`、`employeeNumber`）を残す。

原因は二つの条件の組み合わせである。
`UpdateUser` と `PatchUser` は保存先が返した User をその場で書き換え、manager の解決はその途中で失敗する。
in-memory の `UserRepository.FindBySub` は保存済みの値そのもののポインターを返すので、`Save` を呼ばない失敗でも書き換えが残る。
PostgreSQL の構成は読み込んだ複製だけを変えるので、この食い違いは表に出ない。

修正は `UpdateUser` と `PatchUser` が保存先の User の複製（`editableCopy`）を書き換える形にした。
書き込みがその場で変える参照型の field は `Attributes` だけで、ポインターの field は差し替えるので、`Attributes` を複製する浅い複製で足りる。

| 採った案 | 退けた案 | 退けた理由 |
| --- | --- | --- |
| usecase が複製を書き換える | `UpdateUser` の manager 解決を書き換えの前へ移す | `PatchUser` の複数操作では、失敗しうる操作（userName の重複、manager の解決）が先行操作の後に来るので、順序の入れ替えでは塞げない |
| usecase が複製を書き換える | in-memory リポジトリが複製を返す | 他の Context の利用箇所にも効き、本項目の範囲を越える |

利用者は、切り出しを予定していた時点でこの修正を本項目へ含めると決めた。

### テストの置き場所

新しい観測は `backend/sourcing/scim/handlers_http` の既存ハーネス `newScimHarness` の上に書く。
`backend/shared/http/testing_stack` は SCIM の option を持たない。
既存ハーネスは SCIM の入口を `RegisterRoutes` で建て、保存先を型付きの field として公開しているので、読み直しの観測はそのまま書ける。
実効ロールを観測するため、ハーネスに `groupRepo` と `authenticator` の field を足す。

### readiness pass の結果

| 具体例 | 既存の観測 | 本項目で足す観測 |
| --- | --- | --- |
| `EX-SOURCING-001-01` | `/Users` の絞り込み後の件数とページ | `/Groups` のページと `itemsPerPage` |
| `EX-SOURCING-001-02` | `/Users` の未許可属性と構文不正 | 未許可演算子、`/Groups` の同じ拒否 |
| `EX-SOURCING-001-03` | `/Users` の負の `count` と非整数 `startIndex` | 非整数 `count`、`/Groups` の同じ拒否 |
| `EX-SOURCING-001-04` | 別テナントのトークンによる書き込みの拒否 | コレクション照会の拒否、SCIM 誤り応答、資源を返さないこと |
| `EX-SOURCING-002-01` | 作成、無効化、削除の状態遷移 | なし |
| `EX-SOURCING-002-02` | 別テナントのトークンによる書き込みの拒否 | 失効済み、期限切れ、別テナントのトークンによる作成の拒否と、User が作成されないこと |
| `EX-SOURCING-002-03` | 未対応 `op` の `invalidValue` | 操作要件を満たさない本文の拒否と、User が変わらないこと |
| `EX-SOURCING-002-04` | 存在しない id の GET | 存在しない id の DELETE |
| `EX-SOURCING-003-01` | 省略属性の既定値への復帰 | `meta` の 4 属性 |
| `EX-SOURCING-003-02` | User の `userName` 欠落 | User と Group が変わらないこと、Group の `displayName` 欠落 |
| `EX-SOURCING-003-03` | 応答の `id` の維持 | 本文の `id` でリソースが参照できないこと |
| `EX-SOURCING-004-01` | `name.givenName` の置換 | 対応パスごとに他の属性が変わらないこと |
| `EX-SOURCING-004-02` から `-04` | 拒否の `scimType` | User と Group が変わらないこと、Group の読み取り専用パス |
| `EX-SOURCING-005-01` | メンバーシップの同期、作成時の `type=User` | 実効ロールの更新、PATCH 応答の `type=User` |
| `EX-SOURCING-005-02` | なし | 別テナントの User の追加の拒否と、メンバーシップが作られないこと |
| `EX-SOURCING-005-03` | 作成時の拒否 | PATCH の拒否と、Group が変わらないこと |
| `EX-SOURCING-006-01` | 応答の正規化されたメールアドレス | `User.email` への保存、`/Schemas` の広告範囲 |
| `EX-SOURCING-006-02` | 作成時の複数 `primary` の拒否 | 他の不正形、PUT と PATCH の拒否と、User が変わらないこと |
| `EX-SOURCING-006-03`、`-04` | 投影順序の単体テスト、PATCH の `work` 選択 | 作成時の通信上の順序による選択 |
| `EX-SOURCING-007-01` | 応答と Discovery | `User.Attributes` の 3 キーへの保存 |
| `EX-SOURCING-007-02`、`-03` | 拒否の `scimType`、作成されないこと | `department`、空の `manager`、PUT と PATCH の拒否と、User が変わらないこと。`-03` は欠陥の修正を要した |

### 選んだ実行レシピ

| 触った層 | 実行するタスク |
| --- | --- |
| Go の 1 テスト | `mise run test-go-test -- <package> <test>` |
| Go のパッケージ | `mise run test-go-package -- <package>` |
| 台帳と frontmatter | `mise run check-spec`、`mise run check-work-items` |

## Tasks

- [x] T001 [Acceptance] 25 件を台帳から外し、`mise run check-spec` が未対応 id を名指しで落とすことを確認する。
- [x] T002 [Query] `EX-SOURCING-001-01` から `-04` をコレクション照会のテストへ対応付ける。
- [x] T003 [Lifecycle] `EX-SOURCING-002-01` から `-04` を User のライフサイクルのテストへ対応付ける。
- [x] T004 [Replace] `EX-SOURCING-003-01` から `-03` を PUT のテストへ対応付ける。
- [x] T005 [Patch] `EX-SOURCING-004-01` から `-04` を PATCH のテストへ対応付ける。
- [x] T006 [Group] `EX-SOURCING-005-01` から `-03` を Group 同期のテストへ対応付ける。
- [x] T007 [Attributes] `EX-SOURCING-006-01` から `-007-03` をメールアドレスと Enterprise 拡張のテストへ対応付ける。
- [x] T008 [Fix] 拒否した PUT と PATCH が in-memory 構成で User を書き換える欠陥を、HTTP と usecase の RED を確認してから直す。
- [x] T009 [Verify] 対象パッケージ、仕様検査、総合検証を通して完了記録を作る。

## Verification

- `mise run check-spec` が、`docs/domain/sourcing/scenarios.feature.md` の 25 件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを、消化ごとに観測する。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。減った件数は何も意味しない。注記に「この具体例の何を固定しているか」を書かせることで区別する。親項目の測定では、16 件のうち注記だけで済んだのは 4 件だけだった。
- **`nearby` に分類された件を、テストがある証拠として読む。** 分類が言っているのは「近傍に他の拒否テストがある」だけである。親項目の測定では、`claim-mapping` の 3 件はすべて `nearby` でありながら 2 件はテストが無かった。分類は読む順の材料にとどめる。
- **拒否である具体例のテストが、応答の字面だけを見て書かれる。** 新しく書く拒否テストには [[wi-392-refusal-tests-assert-the-absent-effect]] の規範が効く。注記へ「効果の不在を何で観測したか」を書き、後から区別できるようにする。
- **具体例の `Then` が複数あるのに、観測が 1 つで済まされる。** 親項目の測定では、`EX-WORKLOADIDENTITY-008-01` のようにイベントと実際の効果の双方を言う具体例が複数あった。`Then` の数だけ観測が要る。

## Completion

- **Completed At**: 2026-09-22
- **Summary**:
  `mise run spec-diff` は `main` に対して `no normative specification change` を返した。
  Sourcing が台帳に残していた 25 件の具体例をすべて実装と照合し、`//spec:covers` を付けたテストへ結び付けて台帳から外した。
  既存テストへディレクティブを足すだけで済んだのは `EX-SOURCING-002-01` の 1 件で、他の 24 件は `/Groups` の照会、効果の不在（読み直した資源が拒否の前と同じ、User 数が増えない、メンバーシップが作られない）、保存先の値、有効ロールのいずれかの観測を足した。
  `EX-SOURCING-007-03` の照合で、`PERSISTENCE=memory` の構成では拒否した UpdateScimUser と PatchScimUser が User を途中まで書き換える欠陥が分かった。
  利用者の判断で本項目で直し、`UpdateUser` と `PatchUser` が保存先の User の複製を書き換える形にした。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`、`TestScimEnterpriseExtension_RefusesAnUnresolvableManagerAndLeavesTheUserUnchanged`
  - **Requirement**: REQ-SOURCING-007
  - **Observed Failure**: 台帳から 25 件を外すと、`check-spec` は `EX-SOURCING-001-01` から `EX-SOURCING-007-03` までの 25 件を「declared, but no test names it」で名指しして落ちた。修正前の HTTP テストは `PUT_unknown` で拒否後の User から `emails` が消え、`PATCH_unknown_after_another_operation` で `active` が `false` になり `emails` が消えて落ちた。
  - **Detection Reason**: 台帳の行を消すだけでは検査を通らず、各 id に観測を伴うテストが要る。HTTP テストは拒否応答だけでなく拒否の前後の資源表現全体を比べるので、拒否を返しながら状態を変える実装と区別できる。
- **Unit RED Evidence**:
  - **Test**: `TestWriteUserRefusedByAnUnresolvableManagerLeavesTheStoredUserUnchanged`（`backend/sourcing/scim/usecases`）
  - **Requirement**: REQ-SOURCING-007
  - **Observed Failure**: 修正前は `UpdateUser` と `PatchUser` の両方で、保存先の User の状態が `disabled` になり、email が書き換わって落ちた。
  - **Detection Reason**: `Save` を呼ばない失敗経路で保存先の値を読み直すので、検証の完了より前に保存先の値を書き換える実装を検出する。対応付けだけの 24 件は新しい単体ロジックを持たないので、代替の RED は `mise run check-spec` である。
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/sourcing/scim/usecases` は 146 件の変異のうち 130 件を検出した。生き残った 14 件は `groups.go` と `users.go` の既存の分岐（フィルター属性の射影、応答の組み立て）にあり、今回変えた `editableCopy` とその呼び出しには生存変異がない。
  変異器が表現できない故障として、`editableCopy` が `Attributes` を複製しない形を手で注入し、`TestWriteUserRefusedByAnUnresolvableManagerLeavesTheStoredUserUnchanged/PatchUser` が `Attributes[employee_number]` の残存で落ちることを確認した。`editableCopy` の呼び出しを外す故障は、修正前の状態として Acceptance RED と Unit RED で観測済みである。
- **Verification Results**:
  - `mise run check-spec` - 成功
  - `mise run test-go-package -- ./backend/sourcing/scim/handlers_http` - 成功
  - `mise run test-go-package -- ./backend/sourcing/scim/usecases` - 成功
  - `mise run test-go-changed` - 成功
  - `mise run lint-go` - 成功
  - `mise run verify` - 成功
