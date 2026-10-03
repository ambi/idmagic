---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-03
priority: p1
depends_on: [wi-43431-restructure-specification-documents-by-feature-and-design, wi-43603-repeat-the-design-topics-at-every-level-of-the-specification-tree]
change_kind: docs
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: IdManagement の仕様文書の配置と書式だけを変える。製品の利用者が観測する振る舞い、API、設定は変わらない。
  references: []
initial_context:
  specification:
    - SPECIFICATION_FORMAT.md
    - docs/domain/identity-management/README.md
    - docs/domain/identity-management/glossary.md
    - docs/domain/identity-management/principals/user/README.md
    - docs/domain/identity-management/principals/user/examples.feature.md
  typespec: []
  source:
    - backend/idmanagement/user/usecases/admin_users.go
    - backend/idmanagement/user/handlers_http/admin_user_handler.go
  tests: [tools/check/src/feature-layout.acceptance.test.ts]
  stop_before_reading: [frontend, spec]
spec_impact:
  kind: none
  reason: "IdManagement の文書の置き場所と書式だけを変える。REQ-IDMANAGEMENT-001 から 078 と、その EX の ID、例のステップ、規則が定める値、状態遷移の表は変えない。"
---

# IdManagement の仕様と設計を、機能仕様と内部設計の構造へ移す

## 動機

wi-43431 で、仕様文書を機能仕様と内部設計を軸にした構造へ改める。
最初に移行するのは IdManagement である。
IdManagement には規則が 78 件あり、wi-96676 で書き起こした直後で内容が最も新しい。
しかも問題（判断の箇条書き、内部設計の欠け、ルートに溜まった記述）が最もはっきり現れている。

## 対象範囲

- 機能群と機能ノードを次の構成にする。
  見本の確認の段階で、名前と分け方を確定する。

  | 機能群 | 機能 |
  | --- | --- |
  | プリンシパル | user、account、agent |
  | グループ | group、dynamic-group |
  | 一括転送 | user-csv、group-csv、data-export |
  | 共通 | admin-access、roles |

- ルートにある規則、状態遷移、判断、仕組みの説明を、機能ノードか `design/` へ移す。
  ルートには README（責務と境界、モデル、公開する契約、機能の索引）、glossary、`quality.md`、`design/` だけを残す。
  - REQ-IDMANAGEMENT-014 と 025 は admin-access へ移す。
  - REQ-IDMANAGEMENT-032 と 033 は roles へ移す。
  - CSV の規則 REQ-IDMANAGEMENT-034 から 038 は、一括転送の機能群へ移す。
  - エクスポートの規則 REQ-IDMANAGEMENT-039 から 041 と DataExportLifecycle は、data-export へ移す。
  - CSV の往復変換の説明は `design/csv-transfer.md` へ移す。
- 旧形式の規則（REQ-IDMANAGEMENT-001 から 032 の、`Primary actor:` と説明の段落の形）を、規則文と担保手段の欄の形へ書き直す。
  例は ID とステップを変えずに付録へ移す。
- `decisions.md` の各項目を、規則文への統合、**判断** の注記、`design/decisions.md` の判断のいずれかに振り分ける。
- `internals.md` を、機能の `design.md` または `design/` の話題ごとのファイル（`architecture.md`、`data.md`、`reliability.md`、`performance.md`、`risks.md`）と横断的概念（`design/<concept>.md`）へ移し、表と箇条書きで構造化する。
- `backend/idmanagement` の構成要素を棚卸しし、`design/architecture.md` の構成要素の表と、`design/README.md` と各機能の `design.md` の話題の索引を埋める。
- 各機能仕様の操作の節を、一つの H3 に一つの操作、要約表、正常から拒否への規則の順に整える。
- 移した文書を指すリンクとコメントを直す。

## 対象外

- 規則の内容の変更。
  書き直しの途中で疑わしい挙動を見つけたら、**要判断** の欄に残すか、別の work item を起票する。
- wi-97149 が扱う要判断の決着。

## 設計

### 見本で確かめてから全体へ広げる

最初に user を見本として作る。
作るのは、仕様本文、章（lifecycle）、design.md、付録である。
同時に `design/README.md` と、`design/decisions.md` の判断を 2、3 件作る。
render-docs で HTML を生成して利用者に確認してもらい、その結果を受けてから残りの機能を移す。
構造の誤りを、全機能へ広げた後に見つけると、直す範囲が大きくなるからである。

### 判断の振り分け

| 現行の項目の性質 | 移す先 |
| --- | --- |
| 規則の言い換え | 規則文へ統合する |
| 規則またはモデルの理由 | その規則またはモデルの **判断** の注記 |
| 代替案を比べた重要な判断 | `design/decisions.md` |

`design/decisions.md` へ書く判断の候補は、次のとおりである。

- `User`、`Group`、`Agent` を機能ごとの縦割りのスライスで構成する。
- この Context を `Core` に分類する。
- CSV の適用を、プレビューで保存したペイロードに束縛する。
- 予約ロールを、書き込みが新しく加える分だけで判定する。
- User の削除を、物理削除ではなく Tombstone で行う。
- `Agent` を、`OAuth2Client` に束縛する第 3 のプリンシパルとする。
- CSV に現れない `Group` を削除しない。

代替案は、git の履歴と完了した work item から復元する。
復元できない代替案は書かない。

### 内部設計の網羅

構成要素は、機能スライス（user、group、agent、ルート）ごとの責務、対応する機能仕様、依存先の表と、層の表に分ける。
スライスと層の格子の各セルに三つの項目を書くと読めない量になることを、見本の確認で確かめた。

現行の文書に記述がなく、書き足しが見込まれる視点は次のとおりである。

- Agent の資格情報の束縛
- 動的グループの全件再評価のジョブ
- エクスポートのジョブの流れ
- メールアドレスの変更の確認トークン
- 監査イベントとアウトボックスへの発行
- 永続化と排他

## 計画

1. 見本（user と `design/` の一部）を作り、HTML を利用者に確認してもらう。
2. 残りの機能ノードを移し、ルートの記述を振り分ける。
3. `design/` を完成させ、話題の索引を埋める。
4. リンクとコメントを直し、旧形式の一覧から identity-management を外す。

## タスク

- [x] T001 [Spec] 見本（user、`design/README.md`、`design/decisions.md` の一部）を作り、利用者の確認を受ける。
  N/A: 製品の規範 ID はない。最初の見本の確認で構造の見直しが要ると分かり、wi-43603 で規約を改めてから見本を組み直し、利用者の確認を受けた。`mise run render-docs` が見本を生成し、規則 15 件、未決事項 6 件、操作 9 件が生成した一覧に現れることを確かめた。
- [x] T002 [Spec] 残りの機能ノードと、新設する機能ノード（admin-access、roles、data-export）を作る。
  N/A: 製品の規範 ID はない。新設したノードは admin-access、roles、csv-transfer、data-export の 4 つである。CSV の共通の規則（034〜038）は機能群に宣言できないので、機能ノード csv-transfer に置いた。`mise run check-spec` が「など」を含む規則文を RED として検出し、直して GREEN にした。旧ファイルとの照合の道具（例のステップ、規則の本文、規則の題名、状態表）で、移した内容が欠けていないことを確かめた。
- [x] T003 [Spec] 判断と仕組みの説明を振り分け、`design/` を完成させる。
  N/A: 製品の規範 ID はない。旧 `decisions.md` の 14 項目と各ノードの判断を、規則の判断の欄、モデルの判断、`design/decisions.md`（判断 7 件）に振り分けた。操作ごとの割り当ての列挙（管理 API の権限とスコープ）は TypeSpec の契約データの言い換えなので写さなかった。旧内部設計のうち、コードと食い違う記述（カスケードのトランザクション、Valkey、Agent の所有者の伝播と Group の所有者）はコードに合わせて書き、食い違いを `design/risks.md` に載せた。
- [x] T004 [Spec] 流入するリンク、Go のコメント、未完了の work item のパスを直す。
  N/A: 製品の規範 ID はない。`mise run check-work-items` が旧パスの `affected_spec` を、`mise run check` が旧パスへのリンクを RED として検出した。未完了の作業項目のパスは規則 ID ごとの移動先へ書き換え、完了した作業項目は `tools/check/relocated-spec-paths.json` で読み替えた。work item のスキーマが機能群の下の機能ノードの深さを拒否していたので、`affected_spec.path` の型に一段を加えた。リリース文書と、完了した wi-463 の本文のリンクは、リンクの検査が解決を求めるので新しい場所へ直した。wi-463 の `affected_spec` は書き換えていない。
- [x] T005 [Verify] 検査と spec-diff で、規則が変わっていないことを確かめる。
  `mise run spec-diff` は main に対する規範の差分を報告しない。規範の件数（標準 158、規則 387、例 934、テストが引く ID 1091）は移行の前後で同じである。IdManagement の規則 78 件の題名、例 215 件の 967 ステップ、旧形式でない規則の本文 241 行、状態表 3 つを main の旧ファイルと照合し、すべて一致した。`mise run check`、`mise run render-docs`、`mise run verify` が通る。

## 検証

- `mise run check-spec`、`mise run check`
- `mise run spec-diff` で、REQ と EX の削除と変更が 0 件であることを確かめる。
- 全 EX がテストから引き続き被覆されていることを確かめる。
- 生成した HTML で、規則カード、規則一覧、未決事項、機能地図、状態図を確かめる。

## リスク

旧形式の規則を書き直すときに、意味が変わる可能性がある。
規則文は、付録へ移す例と、規則を引くテストの両方と照らし合わせて書く。

## 完了

- **Completed At**: 2026-10-03
- **Summary**:
  `mise run spec-diff` は、main に対する規範仕様の差分を報告しない。
  IdManagement の仕様と設計を、wi-43431 と wi-43603 が定めた構造へ移した。
  機能群は、プリンシパル（ユーザー、アカウントのセルフサービス、エージェント）、グループ（グループ、動的グループ）、一括転送（CSV の転送、ユーザー CSV、グループ CSV、データエクスポート）、共通（管理 API の認可、ロール）の 4 つで、機能ノードは 11 である。
  Context のルートには、仕様の README（責務と境界、モデル、公開する契約、機能の索引）、用語集、品質要件、設計 `design/` だけを残した。
  設計は、話題ごとのファイル（アーキテクチャ、データ、性能、信頼性、リスク）、横断的概念（CSV の往復変換、イベントと監査の記録）、重要な判断 7 件からなり、各機能ノードの `design.md` も話題の索引を持つ。
  旧形式の規則 31 件を、例とテストに合わせて規則文と担保手段の形へ書き直した。各操作には要約表を置き、規則を正常、状態による分岐、拒否の順に並べた。
  旧文書の記述のうち実装と食い違うもの（User の完全削除のカスケード、データエクスポートの期限切れ、Agent の所有者）は、実装に合わせて設計に書き、食い違いを `design/risks.md` に載せた。規則と状態表は変えていない。監査の記録の経路が二通りあること（保存の後に追記する配信点と、CSV の行と同じトランザクション）と、Aggregate の更新が後勝ちであることも、コードから読み取って設計とリスクに書いた。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（identity-management を旧形式の一覧から外した状態の文書の検査）
  - **Requirement**: N/A: 文書の配置と書式の移行であり、製品の規範を変えない。
  - **Observed Failure**: 見本だけを移した段階で、ルートと各ノードに残る旧形式のファイル 19 件が `not a canonical document` で拒否され、旧形式の一覧が `identity-management uses the feature layout now` で拒否された。全ノードの移行の後は、規則文の「など」の一件だけが残った。
  - **Detection Reason**: 新しい形式の Context では旧形式のファイルを置けないので、移し忘れたファイルと、規則の書式の違反が検査に現れる。
- **Unit RED Evidence**:
  - **Test**: `mise run check-work-items`、`mise run check`（リンクの検査）
  - **Requirement**: N/A: 文書の配置と書式の移行であり、製品の規範を変えない。
  - **Observed Failure**: 完了した作業項目と未完了の作業項目 15 件の `affected_spec` が旧パスで解決できず、リリース文書 5 件と完了した作業項目 1 件のリンクが旧パスを指して失敗した。
  - **Detection Reason**: 規則と文書を移すと、旧パスを指す参照は解決できなくなり、参照の検査がそれを検出する。
- **Change-Resistance Results**:
  文書の移行であり、変異の対象となるコードの変更はない。
  移行の忠実さは、作業用の照合の道具で確かめた。例のステップ、規則の本文、規則の題名、状態表を、main の旧ファイルから抜き出して新しいファイルと比べた。道具の正しさは、先に手で移した user の見本で一致を確かめてから使った。
- **Verification Results**:
  - `mise run check-spec` - 成功
  - `mise run check` - 成功
  - `mise run test-tools` - 成功（807 件）
  - `mise run render-docs` - 成功
  - `mise run verify` - 成功
