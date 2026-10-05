---
status: completed
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-03
priority: p1
depends_on: [wi-96676-transcribe-implicit-specifications-of-identity-management]
change_kind: bugfix
evidence_policy: risk-based-v4
documentation_impact:
  level: release_note
  reason: 行数または項目長の上限を超える CSV のインポートが、先頭の行だけを取り込むプレビューを作らず、投入の時点で拒否されるようになる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-95422-reject-csv-imports-that-exceed-the-row-or-field-limit.md }
initial_context:
  specification:
    - docs/domain/identity-management/user-csv/README.md#REQ-IDMANAGEMENT-004
    - docs/domain/identity-management/group-csv/README.md#REQ-IDMANAGEMENT-026
    - docs/domain/identity-management/group-csv/README.md#REQ-IDMANAGEMENT-029
    - docs/domain/identity-management/user-csv/acceptance.feature.md
    - docs/domain/identity-management/group-csv/acceptance.feature.md
  typespec: []
  source:
    - backend/idmanagement/domain/csv.go
    - backend/idmanagement/user/usecases/user_import.go
    - backend/idmanagement/user/usecases/user_import_planner.go
    - backend/idmanagement/user/handlers_http/admin_user_import_handler.go
    - backend/idmanagement/group/usecases/group_import.go
    - backend/idmanagement/group/usecases/group_membership_import.go
    - frontend/src/features/admin-users/AdminUserImportPage.tsx
    - frontend/src/features/admin-users/AdminUserImportResult.tsx
  tests:
    - backend/idmanagement/user/usecases/user_import_test.go
    - backend/idmanagement/user/usecases/user_import_examples_test.go
    - backend/idmanagement/group/usecases/group_import_examples_test.go
    - backend/idmanagement/group/usecases/group_membership_import_test.go
  stop_before_reading: [backend/jobs, backend/idmanagement/db_postgres, spec]
affected_spec:
  - { path: docs/domain/identity-management/user-csv/README.md, requirement: REQ-IDMANAGEMENT-004, impact: conforms }
  - { path: docs/domain/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-026, impact: conforms }
  - { path: docs/domain/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-029, impact: conforms }
primary_use_cases:
  - id: user-import-submission-over-row-limit
    requirement: REQ-IDMANAGEMENT-004
    observable_result: max_rows を 1 にした User のインポートへ 2 行の CSV を投入すると、too_many_rows で拒否され、プレビューのジョブも成果物も作られない。
    boundary: acceptance
    test: { path: backend/idmanagement/user/usecases/user_import_test.go, name: TestStartUserImportPreviewRefusesFilesBeyondTheRowAndFieldLimits, task: test-go-race }
    fault_model: 投入が byte 数だけを数え、行数と項目長の超過をジョブの中の 1 件のエラーへ後回しにする。
  - id: user-import-apply-over-limit
    requirement: REQ-IDMANAGEMENT-004
    observable_result: プレビューの後に上限が下がり、保存したファイルが上限を超えた場合、適用のジョブは失敗し、先行する行の User を作らない。
    boundary: acceptance
    test: { path: backend/idmanagement/user/usecases/user_import_test.go, name: TestUserImportJobFailsWithoutApplyingAFileBeyondTheLimits, task: test-go-race }
    fault_model: 適用のジョブが上限の超過を解析の途中で検出し、それまでの行を確定してから 1 件のエラーとして成功する。
  - id: group-import-submission-over-row-limit
    requirement: REQ-IDMANAGEMENT-026
    observable_result: 上限を超える Group の CSV の投入が too_many_rows または field_too_large で拒否され、プレビューのジョブが作られない。
    boundary: acceptance
    test: { path: backend/idmanagement/group/usecases/group_import_examples_test.go, name: TestStartGroupImportPreviewRefusesFilesBeyondTheRowAndFieldLimits, task: test-go-race }
    fault_model: Group の投入だけが共有の検査を通らず、byte 数だけを数える。
  - id: membership-import-submission-over-row-limit
    requirement: REQ-IDMANAGEMENT-029
    observable_result: 上限を超えるメンバーシップの CSV の投入が too_many_rows または field_too_large で拒否され、プレビューのジョブが作られない。
    boundary: acceptance
    test: { path: backend/idmanagement/group/usecases/group_membership_import_test.go, name: TestStartGroupMembershipImportPreviewRefusesFilesBeyondTheRowAndFieldLimits, task: test-go-race }
    fault_model: メンバーシップの投入だけが共有の検査を通らず、byte 数だけを数える。
---

# 行数または項目の上限を超える CSV のインポートを、一部も適用せずに拒否する

## 動機

EX-IDMANAGEMENT-004-02、EX-IDMANAGEMENT-026-02、EX-IDMANAGEMENT-029-02 は、CSV が実効 `CsvTransferPolicy` の `max_bytes`、`max_rows`、`max_field_bytes` のいずれかを超えると、インポートの投入を拒否すると定める。
実装が投入の時点で同期的に拒否するのは `max_bytes` だけである。
`max_rows` と `max_field_bytes` は、プレビューのジョブの中で解析が上限に達した位置で初めて検出される。
ジョブはその位置までの行を計画したうえで、上限の超過を一件のエラーとして記録し、成功として終わる。

IdManagement の既存コードを書き起こしたとき、`max_rows` を 1 にした User のインポートで 2 行を投入し、プレビューが成功して `created_rows=1`、`error_total=1` を返すこと、そのプレビューの適用が 1 行目の User を作成することを観測した。
上限の超過は、利用者には「ファイルが拒否された」ではなく「先頭の行だけが取り込まれた」として現れる。
Group とメンバーシップのインポートのジョブも、同じ分岐で CSV のエラーを記録して成功する。

## 対象範囲

- `max_rows` と `max_field_bytes` の超過を、`max_bytes` と同じくプレビューの投入の時点で拒否し、プレビューのジョブを作らない。
- 適用のジョブも、保存したファイルが実効の上限を超えるときは一行も確定せずに失敗させる。
- User、Group、メンバーシップの三つのインポートで同じ契約にする。

## 対象外

- 上限のデフォルト値の変更。
- 見出しの誤り（`invalid_header`）と CSV の構文の誤り（`invalid_csv`）の扱い。これらは今もプレビューのジョブの中で一件のエラーとして記録する。構文の誤りが途中の行にあるファイルでは、その前の行が計画される。

## 設計

### 判定の置き場所

上限の判定は `idmanagement/domain` の CSV 基盤に一つだけ置き、投入とジョブの両方から使う。

| 操作 | シグネチャ | 役割 |
| --- | --- | --- |
| 上限の検査 | `CheckCSVLimits(input io.Reader, policy CSVTransferPolicy) error` | ファイル全体を読み、byte 数、行数、項目長のどれかを超えれば `csv_too_large`、`too_many_rows`、`field_too_large` の `*CSVError` を返す。見出しの語彙は問わず、構文の誤りでは読むのをやめて `nil` を返す |
| 上限内の複製 | `CopyCSVWithinPolicy(output io.Writer, input io.Reader, policy CSVTransferPolicy) error` | 入力を一度だけ読み、`CheckCSVLimits` に通しながら `output` へ書く。投入が成果物ストアへ書き込む関数の中で使い、超過を返せば成果物を作らない |

`CheckCSVLimits` は `CSVReader` と同じ解析器を、見出しの語彙の検査を外した形で使う。
上限の規則を二か所に書くと、片方だけが変わったときに投入とプレビューの判定が食い違うためである。
語彙の検査を外すのは、見出しの誤りを投入で拒否する変更をこの作業に含めないためである。

### 投入

`StartUserImportPreview`、`StartGroupImportPreview`、`StartGroupMembershipImportPreview` は、成果物ストアへ書き込む関数の中の `io.Copy` を `CopyCSVWithinPolicy` に置き換える。
超過は今の `csv_too_large` と同じ経路で `*CSVError` として返り、HTTP の層は 400 と安定コードで応える。
この変換は三つのハンドラーにすでにあり、画面も `too_many_rows` と `field_too_large` の文言を持つので、HTTP の層と画面は変えない。

### ジョブ

三つのジョブのハンドラーは、計画または適用を始める前に、保存したファイルを `CheckCSVLimits` で読み通し、超過ならエラーを返してジョブを失敗させる。
投入で拒否するので、プレビューのジョブでこの検査が働くのは、投入の後に実効の上限が下がった場合だけである。
適用のジョブでは、プレビューの後に上限が下がった場合に、先行する行を確定してから超過に気付くことを防ぐ。
失敗したプレビューは適用できない（`preview_not_ready`）。
ファイルを二度読むが、上限が 64 MiB で、確定より前に全体を知る方法がほかにないため、この費用を受け入れる。

### 実装で決めたこと

| 場所 | 決定 | 理由 |
| --- | --- | --- |
| `CopyCSVWithinPolicy` の書き込みの失敗 | 検査の結果より先に、成果物ストアへの書き込みの失敗を返す | 解析器は下位の読み取りの失敗を構文の誤りと区別せずに返すので、検査の結果だけでは書き込みの失敗が消える |
| `CopyCSVWithinPolicy` の最後の byte 数の確認 | 検査が構文の誤りで読むのをやめた後の残りも複製し、全体の byte 数で上限を確かめる | 少しずつ届く入力では、検査が止まった時点でまだ上限に達していないことがある。変異テストの生き残りから見つけた |
| 投入の `csv_too_large` の行番号 | 解析器が報告する位置（先読みの都合で多くは行 1）をそのまま返す | HTTP の層はコードだけを返し、行番号は利用者に見えない |

### 採用しない案

| 案 | 採用しない理由 |
| --- | --- |
| 投入では拒否せず、プレビューのジョブを失敗させるだけにする | 具体例は投入の拒否と安定コードを求める。失敗したジョブの公開表示はエラーコードを運ばないので、画面は「インポート処理が失敗しました」としか示せない |
| プレビューのジョブを成功させ、行を計画せずにファイル全体のエラーだけを記録する | 成功したプレビューは適用でき、上限を超えたファイルの適用を始められてしまう |
| 投入で成果物を保存した後に読み直して検査する | 拒否したファイルの成果物がストアに残る |

## 計画

1. 投入とジョブの、上限の超過に触れない振る舞い（見出しの誤りと構文の誤りの扱い、上限内のファイルの保存）を特性化テストで固定する。
2. 三つのインポートで、上限を超えるファイルの投入と適用のテストを書き、RED を確かめる。
3. 上限の検査を CSV 基盤に置き、投入とジョブから使う。

## タスク

- [x] T001 [Characterize] 投入とジョブの、上限に触れない振る舞いを特性化テストで固定する。`TestCharacterize{User,Group,GroupMembership}ImportPreviewOfMalformedFiles` と `TestCharacterizeStart{User,Group,GroupMembership}ImportPreviewByteLimit`。投入の byte 数の確認と、ジョブの `RejectedRows++` への変異 12 件を検出することを確かめた。
- [x] T002 [Acceptance] 上限を超えるファイルの投入と適用のテストを三つのインポートについて書き、RED を確かめる。投入は、ジョブを作って成功する（`job=&{… queued …} err=<nil>`）ことで、適用のジョブは `err=<nil>` で成功することで失敗した。ドメインの `TestCheckCSVLimitsReportsOnlyTheLimitTheFileExceeds` と `TestCopyCSVWithinPolicyWritesTheWholeFileOrRefusesIt` は、関数がなくビルドできなかった。
- [x] T003 [App] 上限の超過を投入とジョブでファイル全体の拒否にする。
- [x] T004 [Verify] 変更を検証する。変異テストの生き残りから、構文の誤りの後に残りが上限を超える経路のテスト `TestCopyCSVWithinPolicyCountsTheBytesAfterASyntaxError` を足した。

## 検証

- 各 RED と GREEN：`mise run test-go-test -- <package> <test>`
- 振る舞いが GREEN になった後：`mise run test-go-package -- backend/idmanagement/domain`、`backend/idmanagement/user/usecases`、`backend/idmanagement/group/usecases`、`mise run lint-go`
- 変更への耐性：`mise run test-go-mutation -- backend/idmanagement/domain` と、変更した usecases のパッケージ
- `mise run verify`

## リスク

- 投入で全体を解析するので、投入の応答が解析の時間だけ遅くなる。上限は 64 MiB であり、プレビューのジョブが同じ解析を行う時間と同じ程度である。

## 完了

- **Completed At**: 2026-10-06
- **Summary**:
  `mise run spec-diff -- main` の結果は「no normative specification change against main」であり、規範の差分はない。実装を EX-IDMANAGEMENT-004-02、EX-IDMANAGEMENT-026-02、EX-IDMANAGEMENT-029-02 に合わせた。
  User、Group、メンバーシップの CSV のインポートは、`max_rows` と `max_field_bytes` の超過を `max_bytes` と同じく投入の時点で拒否し、400 と `too_many_rows` または `field_too_large` を返す。プレビューのジョブも成果物も作らない。
  上限の判定は `CSVReader` と一つの実装で共有し、ドメインの `CheckCSVLimits` と `CopyCSVWithinPolicy` が担う。投入は入力を一度だけ読み、検査しながら成果物ストアへ書く。
  三つのジョブは、計画または確定の前に保存したファイル全体を実効の上限で確かめ、超過ならジョブを失敗させる。投入の後に上限が下がった場合も、適用のジョブは先行する行を確定しない。
  見出しの誤りと構文の誤りの扱いは変えず、プレビューのジョブの中で一件のエラーとして記録する。
- **Primary Use Case Evidence**:
  - id: user-import-submission-over-row-limit
    red: 実装前に TestStartUserImportPreviewRefusesFilesBeyondTheRowAndFieldLimits が、行数と項目長の両方の事例で、プレビューのジョブを作って `err=<nil>` を返して失敗した。
    fault_injection: 投入の関数 `StartUserImportPreview` の `CopyCSVWithinPolicy` を `io.Copy` に戻すと、同じテストが同じ失敗をした。
  - id: user-import-apply-over-limit
    red: 実装前に TestUserImportJobFailsWithoutApplyingAFileBeyondTheLimits が `err=<nil>, want the apply job failed with too_many_rows` で失敗した。
    fault_injection: ジョブのハンドラーが `CheckStoredCSVLimits` に実効の上限ではなく既定の上限を渡すと、同じテストが同じ失敗をした。
  - id: group-import-submission-over-row-limit
    red: 実装前に TestStartGroupImportPreviewRefusesFilesBeyondTheRowAndFieldLimits が、プレビューのジョブを作って失敗した。適用の TestGroupImportJobFailsWithoutApplyingAFileBeyondTheLimits も `err=<nil>` で失敗した。
    fault_injection: 投入の関数 `StartGroupImportPreview` の `CopyCSVWithinPolicy` を `io.Copy` に戻すと投入のテストが、ジョブの上限を既定の上限に差し替えると適用のテストが失敗した。
  - id: membership-import-submission-over-row-limit
    red: 実装前に TestStartGroupMembershipImportPreviewRefusesFilesBeyondTheRowAndFieldLimits が、プレビューのジョブを作って失敗した。適用の TestGroupMembershipImportJobFailsWithoutApplyingAFileBeyondTheLimits も `err=<nil>` で失敗した。
    fault_injection: 投入の関数 `StartGroupMembershipImportPreview` の `CopyCSVWithinPolicy` を `io.Copy` に戻すと投入のテストが、ジョブの上限を既定の上限に差し替えると適用のテストが失敗した。
- **Change-Resistance Results**:
  `mise run test-go-mutation` の結果は、`backend/idmanagement/domain` が 63 件検出・9 件生存、`backend/idmanagement/usecases` が 51 件検出・6 件生存、`backend/idmanagement/user/usecases` が 341 件検出・32 件生存、`backend/idmanagement/group/usecases` が 447 件検出・48 件生存だった。
  変更した部分の生き残りは `CopyCSVWithinPolicy` の 2 件（残りの複製の後の `err != nil` の反転、最後の byte 数の比較の `>=`）だけだった。構文の誤りで検査が止まった後に残りが上限を超える経路のテストがなかったためで、TestCopyCSVWithinPolicyCountsTheBytesAfterASyntaxError を足し、2 件とも検出することを確かめた。
  ほかの生き残りは、変更していないポリシーの検証、列名の補完、書き出し器、エラーのページの書き出しと読み出し、結果の欄の写しにある。
  配線の除去は、上の各 `fault_injection` に記録した 6 件を手で注入し、すべて検出した。
  特性化テストは、変更前に投入の byte 数の確認とジョブの `RejectedRows++` への変異 12 件を検出することを確かめ、変更後もすべて通る。
  変更後の分類は次のとおりである。投入の byte 数の境界は EX-IDMANAGEMENT-004-02、026-02、029-02 の `csv_too_large` そのものなので (a) とし、`TestStart{User,Group,GroupMembership}ImportPreviewRefusesFilesBeyondTheByteLimit` へ名前を変えて `//spec:covers` を付けた。禁止した見出しと構文の誤りをプレビューのジョブの中で一件のエラーとして記録する振る舞いは、この作業の対象外なので判断を後に回し、`TestCharacterize*` のまま残した。
  手で解析する入力はなく（`encoding/csv` を使う）、ファジングの対象は足していない。
- **Verification Results**:
  - `mise run test-go-changed` - 成功
  - `mise run lint-go` - 成功
  - `mise run spec-diff -- main` - 規範の差分なし
  - `mise run verify` - 成功。初回は `check-repository` が、`impact: conforms` の要件 ID を名指すテストがないことで失敗したので、`//spec:covers` の先頭に要件 ID を加えた
  - `mise run test-ui-e2e` - 成功（39 件）
