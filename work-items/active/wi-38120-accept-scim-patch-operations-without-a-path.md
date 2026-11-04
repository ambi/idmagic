---
status: pending
authors: [tn]
risk: medium
created_at: 2026-11-05
priority: p1
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/sourcing/standards.md, requirement: RFC7644-PATCH }
  - { path: docs/modules/sourcing/features/scim.md, requirement: REQ-SOURCING-004 }
---

# `path` を省いた SCIM の PATCH の `add` と `replace` を受け入れる

## 動機

RFC 7644 §3.5.2.1 と §3.5.2.3 は、`add` と `replace` の `path` を OPTIONAL とし、省いたときは `value` を属性の名前と値の組として対象のリソースへ適用すると定める。
Microsoft Entra ID のプロビジョニングは、利用者の無効化を `{"op":"Replace","value":{"active":false}}` のように `path` なしで送る。
`ParseUserPatchOps` と `ParseGroupPatchOps` は、`path` が空の操作を `invalidPath` で拒否する。
[[wi-77372-inventory-standards-acceptance-rules]] の棚卸しで見つけた。

## 対象範囲

- User と Group の PATCH で、`path` のない `add` と `replace` の `value` のオブジェクトを、属性ごとの操作へ展開する。展開した各属性には、`path` を指定したときと同じ許可の一覧、読み取り専用の検査、値の型の検査を適用する。
- `value` がオブジェクトでなければ `invalidValue` で拒否する。`path` のない `remove` は RFC 7644 §3.5.2.2 どおり `noTarget` で拒否する。
- Enterprise 拡張の URN をキーにした入れ子の `value` を扱う。
- `standards.md` に受け入れの行を加え、`path` を省いた要求を SCIM の入口へ送る試験を加える。

## 対象外

- `path` の値フィルター（`emails[type eq "work"].value`、`members[value eq "..."]`）。[[wi-248-scim-complex-value-filter-bracket-syntax]] が扱う。

## タスク

- [ ] T001 [Spec] 受け入れの行と REQ-SOURCING-004 の例を加える。
- [ ] T002 [Acceptance] `path` を省いた User の無効化と Group の名前の変更で RED を確かめる。
- [ ] T003 [Domain] 展開を実装し、GREEN にする。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- PATCH の解析の単体試験：展開した操作が既存の検査を通ることを確かめるため。
- SCIM の HTTP の受け入れ試験：User の `active` と Group の `displayName` の変化を観測するため。

## リスク

展開が許可の一覧を迂回すると、読み取り専用の属性を書き換えられる。
展開した結果を既存の `path` の検査へ通す形にし、検査を二重に書かない。
