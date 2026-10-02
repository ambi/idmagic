---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-03
priority: p1
depends_on: [wi-43431-restructure-specification-documents-by-feature-and-design]
change_kind: docs
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
  ルートには README、glossary、`design/` だけを残す。
  - REQ-IDMANAGEMENT-014 と 025 は admin-access へ移す。
  - REQ-IDMANAGEMENT-032 と 033 は roles へ移す。
  - CSV の規則 REQ-IDMANAGEMENT-034 から 038 は、一括転送の機能群へ移す。
  - エクスポートの規則 REQ-IDMANAGEMENT-039 から 041 と DataExportLifecycle は、data-export へ移す。
  - CSV の往復変換の説明は `design/csv-transfer.md` へ移す。
- 旧形式の規則（REQ-IDMANAGEMENT-001 から 032 の、`Primary actor:` と説明の段落の形）を、規則文と担保手段の欄の形へ書き直す。
  例は ID とステップを変えずに付録へ移す。
- `decisions.md` の各項目を、規則文への統合、**判断** の注記、`design/decisions.md` の判断のいずれかに振り分ける。
- `internals.md` を、機能の `design.md` または `design/<concept>.md` へ移し、表と箇条書きで構造化する。
- `backend/idmanagement` の構成要素を棚卸しし、`design/README.md` の構成表と設計視点の網羅表を埋める。
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

構成表は、機能スライス（user、group、agent、ルート）と層（domain、ports、usecases、handlers_http、db_postgres、db_memory）を軸にする。
各セルには、責務、公開するポート、依存先を書く。

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
3. `design/` を完成させ、網羅表を埋める。
4. リンクとコメントを直し、旧形式の一覧から identity-management を外す。

## タスク

- [ ] T001 [Spec] 見本（user、`design/README.md`、`design/decisions.md` の一部）を作り、利用者の確認を受ける。
- [ ] T002 [Spec] 残りの機能ノードと、新設する機能ノード（admin-access、roles、data-export）を作る。
- [ ] T003 [Spec] 判断と仕組みの説明を振り分け、`design/` を完成させる。
- [ ] T004 [Spec] 流入するリンク、Go のコメント、未完了の work item のパスを直す。
- [ ] T005 [Verify] 検査と spec-diff で、規則が変わっていないことを確かめる。

## 検証

- `mise run check-spec`、`mise run check`
- `mise run spec-diff` で、REQ と EX の削除と変更が 0 件であることを確かめる。
- 全 EX がテストから引き続き被覆されていることを確かめる。
- 生成した HTML で、規則カード、規則一覧、未決事項、機能地図、状態図を確かめる。

## リスク

旧形式の規則を書き直すときに、意味が変わる可能性がある。
規則文は、付録へ移す例と、規則を引くテストの両方と照らし合わせて書く。
