---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-03
priority: p2
depends_on: [wi-96797-stop-agents-when-their-owner-stops]
change_kind: docs
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: IdManagement の規則の題名と、規則がどの操作の節に属するかを改め、新しい規則 ID を加える。製品の振る舞い、API、設定は変わらないが、規則 ID と題名を参照する読み手へ知らせる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-54453-split-identity-management-rules-by-operation.md }
initial_context:
  specification:
    - SPECIFICATION_FORMAT.md
    - docs/domain/identity-management/principals/user/lifecycle.md
    - docs/domain/identity-management/principals/agent/README.md
    - docs/domain/identity-management/groups/group/README.md
    - docs/domain/identity-management/bulk-transfer/data-export/README.md
  typespec: []
  source: []
  tests: [backend/idmanagement/handlers_http, backend/idmanagement/user/usecases, backend/idmanagement/group/usecases]
  stop_before_reading: [frontend, spec]
affected_spec:
  - { path: docs/domain/identity-management/groups/group/README.md, requirement: REQ-IDMANAGEMENT-084 }
  - { path: docs/domain/identity-management/groups/group/README.md, requirement: REQ-IDMANAGEMENT-085 }
  - { path: docs/domain/identity-management/bulk-transfer/data-export/README.md, requirement: REQ-IDMANAGEMENT-086 }
  - { path: docs/domain/identity-management/bulk-transfer/data-export/README.md, requirement: REQ-IDMANAGEMENT-087 }
  - { path: docs/domain/identity-management/bulk-transfer/data-export/README.md, requirement: REQ-IDMANAGEMENT-088 }
  - { path: docs/domain/identity-management/principals/account/README.md, requirement: REQ-IDMANAGEMENT-002 }
  - { path: docs/domain/identity-management/principals/account/README.md, requirement: REQ-IDMANAGEMENT-003 }
  - { path: docs/domain/identity-management/bulk-transfer/user-csv/README.md, requirement: REQ-IDMANAGEMENT-004 }
  - { path: docs/domain/identity-management/principals/user/README.md, requirement: REQ-IDMANAGEMENT-005 }
  - { path: docs/domain/identity-management/bulk-transfer/user-csv/README.md, requirement: REQ-IDMANAGEMENT-006 }
  - { path: docs/domain/identity-management/bulk-transfer/user-csv/README.md, requirement: REQ-IDMANAGEMENT-007 }
  - { path: docs/domain/identity-management/bulk-transfer/group-csv/README.md, requirement: REQ-IDMANAGEMENT-008 }
  - { path: docs/domain/identity-management/principals/agent/README.md, requirement: REQ-IDMANAGEMENT-009 }
  - { path: docs/domain/identity-management/principals/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-010 }
  - { path: docs/domain/identity-management/principals/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-011 }
  - { path: docs/domain/identity-management/principals/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-013 }
  - { path: docs/domain/identity-management/common/admin-access/README.md, requirement: REQ-IDMANAGEMENT-014 }
  - { path: docs/domain/identity-management/groups/group/README.md, requirement: REQ-IDMANAGEMENT-015 }
  - { path: docs/domain/identity-management/principals/account/README.md, requirement: REQ-IDMANAGEMENT-016 }
  - { path: docs/domain/identity-management/principals/account/README.md, requirement: REQ-IDMANAGEMENT-017 }
  - { path: docs/domain/identity-management/principals/account/README.md, requirement: REQ-IDMANAGEMENT-018 }
  - { path: docs/domain/identity-management/principals/account/README.md, requirement: REQ-IDMANAGEMENT-019 }
  - { path: docs/domain/identity-management/groups/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-020 }
  - { path: docs/domain/identity-management/groups/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-021 }
  - { path: docs/domain/identity-management/groups/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-022 }
  - { path: docs/domain/identity-management/groups/group/README.md, requirement: REQ-IDMANAGEMENT-024 }
  - { path: docs/domain/identity-management/common/admin-access/README.md, requirement: REQ-IDMANAGEMENT-025 }
  - { path: docs/domain/identity-management/bulk-transfer/group-csv/README.md, requirement: REQ-IDMANAGEMENT-026 }
  - { path: docs/domain/identity-management/bulk-transfer/group-csv/README.md, requirement: REQ-IDMANAGEMENT-027 }
  - { path: docs/domain/identity-management/bulk-transfer/group-csv/README.md, requirement: REQ-IDMANAGEMENT-028 }
  - { path: docs/domain/identity-management/bulk-transfer/group-csv/README.md, requirement: REQ-IDMANAGEMENT-029 }
  - { path: docs/domain/identity-management/bulk-transfer/group-csv/README.md, requirement: REQ-IDMANAGEMENT-030 }
  - { path: docs/domain/identity-management/common/roles/README.md, requirement: REQ-IDMANAGEMENT-032 }
  - { path: docs/domain/identity-management/bulk-transfer/csv-transfer/README.md, requirement: REQ-IDMANAGEMENT-037 }
  - { path: docs/domain/identity-management/bulk-transfer/data-export/README.md, requirement: REQ-IDMANAGEMENT-039 }
  - { path: docs/domain/identity-management/bulk-transfer/data-export/README.md, requirement: REQ-IDMANAGEMENT-041 }
  - { path: docs/domain/identity-management/principals/user/README.md, requirement: REQ-IDMANAGEMENT-042 }
  - { path: docs/domain/identity-management/principals/user/README.md, requirement: REQ-IDMANAGEMENT-043 }
  - { path: docs/domain/identity-management/principals/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-046 }
  - { path: docs/domain/identity-management/principals/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-048 }
  - { path: docs/domain/identity-management/principals/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-049 }
  - { path: docs/domain/identity-management/principals/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-050 }
  - { path: docs/domain/identity-management/principals/account/README.md, requirement: REQ-IDMANAGEMENT-053 }
  - { path: docs/domain/identity-management/principals/account/README.md, requirement: REQ-IDMANAGEMENT-054 }
  - { path: docs/domain/identity-management/bulk-transfer/user-csv/README.md, requirement: REQ-IDMANAGEMENT-057 }
  - { path: docs/domain/identity-management/bulk-transfer/user-csv/README.md, requirement: REQ-IDMANAGEMENT-058 }
  - { path: docs/domain/identity-management/groups/group/README.md, requirement: REQ-IDMANAGEMENT-060 }
  - { path: docs/domain/identity-management/groups/group/README.md, requirement: REQ-IDMANAGEMENT-061 }
  - { path: docs/domain/identity-management/groups/group/README.md, requirement: REQ-IDMANAGEMENT-062 }
  - { path: docs/domain/identity-management/groups/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-065 }
  - { path: docs/domain/identity-management/groups/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-066 }
  - { path: docs/domain/identity-management/groups/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-068 }
  - { path: docs/domain/identity-management/bulk-transfer/group-csv/README.md, requirement: REQ-IDMANAGEMENT-072 }
  - { path: docs/domain/identity-management/principals/agent/README.md, requirement: REQ-IDMANAGEMENT-073 }
  - { path: docs/domain/identity-management/principals/agent/README.md, requirement: REQ-IDMANAGEMENT-074 }
  - { path: docs/domain/identity-management/principals/agent/README.md, requirement: REQ-IDMANAGEMENT-076 }
  - { path: docs/domain/identity-management/principals/agent/README.md, requirement: REQ-IDMANAGEMENT-077 }
  - { path: docs/domain/identity-management/principals/agent/README.md, requirement: REQ-IDMANAGEMENT-078 }
  - { path: docs/domain/identity-management/principals/agent/README.md, requirement: REQ-IDMANAGEMENT-081 }
  - { path: docs/domain/identity-management/principals/agent/README.md, requirement: REQ-IDMANAGEMENT-082 }
---

# IdManagement の規則を操作ごとに切り直し、題名の型をそろえる

## 動機

IdManagement の規則には、一つの規則が複数の操作の義務を抱えるものがある。
たとえば Agent の登録の規則は、資格情報の束縛と、無効化と再有効化の義務も述べていた。
User のエクスポートの規則は、データエクスポートの開始、生成、ダウンロード、取り消しの流れを、User に固有の義務と一緒に述べていた。
読み手は、ある操作の義務を知るために、ほかの操作の節の規則まで探すことになり、操作の要約表と規則の対応も崩れていた。

題名の型もそろっていない。
「管理者は〜できる」という行為者の能力の形と、「〜の正規化」のような名詞句が混ざり、題名だけでは何を保証する規則かがわからない。

## 対象範囲

- `SPECIFICATION_FORMAT.md` に、規則は置いた操作の義務だけを述べること、複数の操作に共通する一つの不変条件だけは一つの規則にしてよいこと、題名の型を定める。
- IdManagement の規則のうち、複数の操作の義務を抱えるものを、操作ごとの規則に分ける。
- IdManagement の規則の題名を、「〈条件〉の〈対象〉の〈操作〉は、〈結果〉」の型にそろえる。
- 移した例を引くテストの `//spec:covers` を付け替える。

## 対象外

- 一つの操作の中で多くの側面を抱える規則（Group の CSV のインポートなど）を、側面ごとに分けること。
- IdManagement 以外の Context の規則。
- 製品の振る舞いの変更。

## 設計

### 分けた規則

| 元の規則 | 分け方 |
| --- | --- |
| REQ-IDMANAGEMENT-009（Agent の登録） | 束縛を REQ-IDMANAGEMENT-074 へ、無効化と再有効化を REQ-IDMANAGEMENT-076 へ移す |
| REQ-IDMANAGEMENT-011（削除の予約と復元） | 復元を REQ-IDMANAGEMENT-049 へ移し、削除の予約の冪等性を REQ-IDMANAGEMENT-048 から受け取る |
| REQ-IDMANAGEMENT-013（完全削除と自分自身の拒否） | 完全削除の正常経路を REQ-IDMANAGEMENT-050 へ移し、三つの操作に共通する自分自身の拒否の不変条件にする |
| REQ-IDMANAGEMENT-046（User の無効化と再有効化） | 記憶済みの端末の失効を REQ-IDMANAGEMENT-010 へ移す |
| REQ-IDMANAGEMENT-024、REQ-IDMANAGEMENT-061（Group の作成と更新） | 更新の義務を REQ-IDMANAGEMENT-062 へ移す |
| REQ-IDMANAGEMENT-015（メンバーの追加と所属グループの参照） | 参照を新しい操作の節と REQ-IDMANAGEMENT-084 に分ける |
| REQ-IDMANAGEMENT-022（規則の保存と手動のメンバー操作） | 手動のメンバー操作の拒否を Group の REQ-IDMANAGEMENT-085 に分ける |
| REQ-IDMANAGEMENT-066（規則の保存、有効化、無効化） | 無効化の版と拒否を REQ-IDMANAGEMENT-068 へ移す |
| REQ-IDMANAGEMENT-020（保存、有効化、再評価） | 全件の再評価の節へ移し、再評価の結果だけを述べる |
| REQ-IDMANAGEMENT-006（User のエクスポート） | 開始を REQ-IDMANAGEMENT-039、生成を新しい操作の節と REQ-IDMANAGEMENT-086、ダウンロードを REQ-IDMANAGEMENT-087、種類とテナントの境界を REQ-IDMANAGEMENT-088、取り消しを REQ-IDMANAGEMENT-041 へ移し、User に固有の列と数式の注入の防止だけを残す |
| REQ-IDMANAGEMENT-017（メールアドレスの変更） | 起票と確定を別の操作の節に分け、起票の義務を REQ-IDMANAGEMENT-053 へ移す |

例は移した先の規則の番号で振り直し、元のテストは新しい例の ID を引く。
完了済みの作業項目が主要ユースケースとして指す規則（REQ-IDMANAGEMENT-009、REQ-IDMANAGEMENT-046 など）は、ID を残し、テストのファイルの「主要ユースケース追跡」で追跡を保つ。

### 共通の不変条件として残す規則

REQ-IDMANAGEMENT-013、REQ-IDMANAGEMENT-060、REQ-IDMANAGEMENT-061、REQ-IDMANAGEMENT-064、REQ-IDMANAGEMENT-077、REQ-IDMANAGEMENT-081、REQ-IDMANAGEMENT-088 は、複数の操作に共通する一つの不変条件なので、一つの規則のまま残す。
ほかの操作の要約表から、規則の ID で参照する。

## 計画

1. 規約を書く。
2. 操作をまたぐ規則を分け、例とテストの引用を移す。
3. 題名をそろえる。
4. リンクを直し、検証する。

## タスク

- [x] T001 [Spec] `SPECIFICATION_FORMAT.md` に規則の単位と題名の型を書く。
- [x] T002 [Spec] 操作をまたぐ規則を分け、例とテストの引用を移す。
- [x] T003 [Spec] 題名をそろえる。
- [x] T004 [Verify] 変更を検証する。
  `mise run check` が、題名の変更で切れた規則のアンカー（設計の指針 1 件、リリース文書 5 件）を検出したので、ページへのリンクか新しい節のアンカーへ直した。
  完了した作業項目 wi-573 の主要ユースケースが、テストのファイルで REQ-IDMANAGEMENT-009 を名指すことを求めたので、そのファイルに主要ユースケースの追跡を加えた。

## 検証

- `mise run check`
- `mise run verify`

## リスク

- 題名の変更で、題名から作るアンカーへのリンクが切れる。`mise run check` が検出するので、ページへのリンクか新しいアンカーへ直す。

## 完了

- **Completed At**: 2026-10-03
- **Summary**:
  `mise run spec-diff -- work-item/wi-96797` は、追加した規則 REQ-IDMANAGEMENT-084 から 088 と、題名または本文を変えた規則 55 件を報告する。
  `SPECIFICATION_FORMAT.md` に、規則は置いた操作の義務だけを述べること、複数の操作に共通する一つの不変条件だけは一つの規則にしてよいこと、題名を「〈条件〉の〈対象〉の〈操作〉は、〈結果〉」の一文にすることを定めた。
  IdManagement の規則のうち、操作をまたいでいた 11 件を操作ごとの規則に分けた。所属グループの参照、エクスポートの生成、メールアドレスの変更の起票と確定は、それぞれ独立した操作の節にした。
  能力の形と名詞句の題名と、分けた規則の題名、合わせて 53 件を、結果を述べる一文にそろえた。
  製品のコードは変えていない。テストは、移した例の ID を引くように `//spec:covers` を付け替えた。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check`
  - **Requirement**: N/A: 規則の単位と題名の整理であり、製品の振る舞いを変えない。
  - **Observed Failure**: 規則の題名を変えた直後に、付録の `Rule` の題名が仕様本文の宣言と異なることと、題名から作るアンカーへのリンクが切れたことを報告した。完了した作業項目 wi-18703 と wi-573 の主要ユースケースが、テストのファイルで規則を名指せなくなったことも報告した。
  - **Detection Reason**: 付録の題名の一致、リンクの解決、主要ユースケースの追跡は、どれも題名と規則の移動に追従しない参照を検出する。
- **Unit RED Evidence**:
  - **Test**: `mise run check`（作業範囲の仕様の差分）
  - **Requirement**: N/A: 規則の単位と題名の整理であり、製品の振る舞いを変えない。
  - **Observed Failure**: 作業項目を加える前は、変えた規則が `affected_spec` にないことを規則ごとに報告した。
  - **Detection Reason**: 作業範囲の仕様の差分は、宣言していない規則の変更を拒否する。
- **Change-Resistance Results**:
  規則の整理であり、変異の対象となるコードの変更はない。
  取りこぼしは、変更前と変更後の規則文と例のステップを行ごとに比べて確かめた。変更前にあり変更後にない行は、どれも移した先の規則か例で言い換えて残っている。
- **Verification Results**:
  - `mise run check` - 成功
  - `mise run verify` - 成功
