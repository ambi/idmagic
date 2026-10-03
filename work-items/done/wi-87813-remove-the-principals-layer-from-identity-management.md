---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-03
priority: p2
depends_on: []
change_kind: docs
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: IdManagement の仕様文書の配置と索引だけを変える。製品の利用者が観測する振る舞い、API、設定は変わらない。
  references: []
initial_context:
  specification:
    - SPECIFICATION_FORMAT.md
    - docs/domain/identity-management/README.md
  typespec: []
  source:
    - tools/workspace/src/document-layout.ts
    - tools/check/src/specification-rules.ts
    - tools/check/relocated-spec-paths.json
    - tools/check/src/spec-diff.ts
  tests: [tools/check/src/spec-diff.test.ts]
  stop_before_reading: [backend, frontend, spec]
spec_impact:
  kind: none
  reason: "IdManagement の機能仕様の配置と機能群の名前を変えるだけで、要件の ID、題名、本文、例、TypeSpec の契約、外部から観測できる振る舞いは変えない。"
---

# IdManagement の機能を `principals` と `common` という抽象的な機能群に入れず、ドメインの語で配置する

## 動機

IdManagement の機能仕様は、`principals`、`groups`、`bulk-transfer`、`common` の四つの機能群に分かれている。
このうち `principals`（User、アカウントのセルフサービス、Agent）と `common`（管理 API の認可、ロール）は、ドメインの語ではなく、機能を分類するために作った抽象語である。
読み手は、User の仕様を探すときに「プリンシパル」という概念を経由する必要があり、どの機能がそこに属するかを名前から推測できない。

## 対象範囲

- `principals`、`common`、`groups`、`bulk-transfer` の四つの機能群を廃止し、11 の機能ノードを Context の直下に並べる。
  機能群の `README.md` が書いていた境界の説明は、Context の `README.md` か各機能ノードへ移す。
- Context の `README.md` の責務と機能の索引から「プリンシパル」という語を除き、User、Group、Agent の台帳として書き直す。
- 移動した要件のパスを `tools/check/relocated-spec-paths.json` に加え、完了した work item の `affected_spec` が解決できるようにする。
- 未完了の work item の `affected_spec` と、文書間のリンクを新しいパスへ更新する。
- `spec-diff` が規則の本文のリンクをラベルだけで比べるようにする。
  移動で相対リンクのパスだけが変わった規則を、仕様の変更として報告させない。

## 対象外

- 要件の本文と書式の変更。
  wi-17076 が扱う。
- コードの機能スライスの名前と構成の変更。
- 完了した work item とリリース文書の本文に残る旧パス。
  履歴なので書き換えず、`affected_spec` は `relocated-spec-paths.json` で解決する。

## 設計

### 配置の案

| 案 | 配置 | 利点 | 欠点 |
| --- | --- | --- | --- |
| A：機能群の階層をなくす | `identity-management/<feature>/` に 11 の機能を並べる | 抽象語が一つも残らない | 仕様の木の検査が機能群の階層を必須としている場合は検査の変更が要る |
| B：機能群をドメインの語で作り直す | `users/`（user、account）、`agents/`（agent）、`groups/`、`bulk-transfer/`、`access/`（admin-access、roles） | 検査の変更が要らない | `agents/agent` のように名前が重なり、`access` も抽象語として残る |

案 A を採る。
`groups` と `bulk-transfer` もドメインの語ではあるが、一部の機能群だけを残すと、Context の直下に機能ノードと機能群が混在する。
機能が十数個の Context では、機能群の階層が見渡しやすさに寄与しない。
機能ノードの名前はコードの機能スライス（`backend/idmanagement/user`、`agent`、`group`）との対応で決まるので、機能群をなくしても対応の判定は変わらない。

文書配置の検査（`documentAllowance`）は、子のディレクトリを持つ段を機能群、持たない段を機能ノードと判定する。
Context の直下の段も機能ノードとして扱われ、`SPECIFICATION_FORMAT.md` も小さな Context で機能群を置かない配置を認めているので、検査の変更は要らない。

`relocated-spec-paths.json` では、旧パスを新しいパスへ対応させる項目を加え、既存の項目の移動先も新しいパスへ書き換える。

### 規則の本文のリンクの比較

`spec-diff` は規則の本文を行ごとにそのまま比べていたので、機能ノードを移すと、相対リンクのパスだけが変わった規則（REQ-IDMANAGEMENT-004、026、032、050）を変更として報告し、`check-repository` が `spec_impact: none` を拒否した。

| 案 | 内容 | 利点 | 欠点 |
| --- | --- | --- | --- |
| A：リンクをラベルで比べる | 本文の正規化で `[ラベル](リンク先)` を `[ラベル]` にする | 文書の移動で誤検知しない | ラベルを変えずにリンク先だけを差し替えた変更を検出しない |
| B：4 件を `affected_spec` に `modifies` として挙げる | 検査を変えない | 変更が要らない | 変わっていない要件を変更したと記録し、文書を移すたびに繰り返す |

案 A を採る。
リンク先は参照の道筋であり、規則が定める内容はラベルと周囲の文が表す。
リンク先の存在は `check-links` が別に検査する。

## 計画

1. 仕様の木の検査が、機能ノードを Context の直下に置くことを許すかを確かめる。
   許さない場合は、機能群の階層を任意にする検査の変更を本項目に含めるか、案 B を採るかを決める。
2. 機能ノードを移動し、`relocated-spec-paths.json` を更新する。
3. Context の `README.md` を書き直し、リンクを更新する。

## タスク

- [x] T001 [Tooling] 仕様の木の検査が機能群の階層をなくした配置を受け付けるかを確かめ、配置の案を決める。
  `documentAllowance` が子を持たない段を機能ノードとして扱うことを読んで確かめ、案 A に決めた。
- [x] T002 [Docs] 機能ノードを移動し、`relocated-spec-paths.json` を更新する。
  RED：N/A: 製品の要件を変えない文書の移動である。代替検査として、移動直後に `mise run check-work-items` と `mise run check-links` が旧パスを報告して失敗することを確かめた。
- [x] T003 [Docs] Context の `README.md` と文書間のリンク、未完了の work item の `affected_spec` を更新する。
  機能群の `README.md` が書いていた境界の説明は、各機能ノードの「扱わないもの」にすでにあったので、Context の機能の索引へ各機能の内容だけを移した。
- [x] T004 [Tooling] `spec-diff` が規則の本文のリンクをラベルだけで比べるようにする。
  RED：`mise run test-tools-file -- check/src/spec-diff.test.ts` の `compares a link in a rule body by its label, not by its relative target` が、相対パスだけを変えた規則を `REQ-DEMO-001` の変更として報告して失敗した。
- [x] T005 [Verify] 変更を検証する。

## 検証

- `mise run check-work-items`
- `mise run spec-diff` で、要件の差分がないこと。
- `mise run check-links`
- `mise run verify`

## リスク

- 移動で、完了した work item の `affected_spec` が解決できなくなる。
  `relocated-spec-paths.json` に旧パスを加え、`mise run check-work-items` で確かめる。
- wi-17076 と同時に進めると、`principals/user` の書き直しと移動が衝突する。
  どちらかを完了してから他方に着手する。

## 完了

- **Completed At**: 2026-10-04
- **Summary**:
  `mise run spec-diff` は main に対して規範の変更なしを報告する。
  IdManagement の 11 の機能ノードを Context の直下へ移し、`principals`、`common`、`groups`、`bulk-transfer` の機能群を廃止した。
  Context の `README.md` は、責務を User、Group、Agent の台帳として書き直し、機能の索引を 11 の機能の表にした。
  旧パスは `relocated-spec-paths.json` で新しいパスへ対応させた。
  `spec-diff` は規則の本文のリンクをラベルだけで比べるようになり、文書の移動で変わる相対パスを仕様の変更として報告しなくなった。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-work-items`、`mise run check-links`
  - **Requirement**: N/A: 要件を変えない文書の移動である。
  - **Observed Failure**: 機能ノードを移した直後、`check-work-items` は未完了の work item の `affected_spec` が旧パスを指すと報告し、`check-links` は 118 件のリンク切れを報告して失敗した。
  - **Detection Reason**: 受け入れ境界は該当しない。移動で解決できなくなる参照とリンクを、この二つの検査が一件ずつ報告する。
- **Unit RED Evidence**:
  - **Test**: `mise run test-tools-file -- check/src/spec-diff.test.ts` の `compares a link in a rule body by its label, not by its relative target`
  - **Requirement**: N/A: 開発ツールの変更であり、製品の要件はない。
  - **Observed Failure**: 相対パスだけを変えた規則を、`REQ-DEMO-001` の変更として報告して失敗した。
  - **Detection Reason**: 同じテストがラベルを変えた規則を変更として報告することも表明するので、リンクを丸ごと比較から外す誤実装も検出する。
- **Change-Resistance Results**:
  低リスクのため変異試験は行っていない。正規化の置換を外した状態で上のテストが失敗することは、RED として観測済みである。
- **Verification Results**:
  - `mise run verify` - 成功
  - `mise run check-repository` - 成功
  - `mise run spec-diff` - 規範の変更なし
