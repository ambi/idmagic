---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-03
priority: p2
depends_on: []
change_kind: docs
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

- `principals` と `common` の機能群を廃止し、配下の機能をドメインの語で配置し直す。
- Context の `README.md` の責務と機能の索引から「プリンシパル」という語を除き、User、Group、Agent の台帳として書き直す。
- 移動した要件のパスを `tools/check/relocated-spec-paths.json` に加え、完了した work item の `affected_spec` が解決できるようにする。
- 未完了の work item の `affected_spec` と、文書間のリンクを新しいパスへ更新する。

## 対象外

- 要件の本文と書式の変更。
  wi-17076 が扱う。
- コードの機能スライスの名前と構成の変更。
- `groups` と `bulk-transfer` の機能群の名前の変更。
  どちらもドメインの語なので残す。

## 設計

### 配置の案

| 案 | 配置 | 利点 | 欠点 |
| --- | --- | --- | --- |
| A：機能群の階層をなくす | `identity-management/<feature>/` に 11 の機能を並べる | 抽象語が一つも残らない | 仕様の木の検査が機能群の階層を必須としている場合は検査の変更が要る |
| B：機能群をドメインの語で作り直す | `users/`（user、account）、`agents/`（agent）、`groups/`、`bulk-transfer/`、`access/`（admin-access、roles） | 検査の変更が要らない | `agents/agent` のように名前が重なり、`access` も抽象語として残る |

案 A を第一とする。
機能が十数個の Context では、機能群の階層が見渡しやすさに寄与しない。
機能ノードの名前はコードの機能スライス（`backend/idmanagement/user`、`agent`、`group`）との対応で決まるので、機能群をなくしても対応の判定は変わらない。

## 計画

1. 仕様の木の検査が、機能ノードを Context の直下に置くことを許すかを確かめる。
   許さない場合は、機能群の階層を任意にする検査の変更を本項目に含めるか、案 B を採るかを決める。
2. 機能ノードを移動し、`relocated-spec-paths.json` を更新する。
3. Context の `README.md` を書き直し、リンクを更新する。

## タスク

- [ ] T001 [Tooling] 仕様の木の検査が機能群の階層をなくした配置を受け付けるかを確かめ、配置の案を決める。
- [ ] T002 [Docs] 機能ノードを移動し、`relocated-spec-paths.json` を更新する。
- [ ] T003 [Docs] Context の `README.md` と文書間のリンク、未完了の work item の `affected_spec` を更新する。
- [ ] T004 [Verify] 変更を検証する。

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
