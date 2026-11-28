---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-11-28
priority: p1
group: scim-provisioning
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/sourcing/standards.md, requirement: RFC7643-CORE-RESOURCES }
  - { path: spec/modules/sourcing/main.tsp, symbol: IdMagic.Contract.CreateScimUser }
  - { path: spec/modules/sourcing/main.tsp, symbol: IdMagic.Contract.UpdateScimUser }
---

# SCIM の User の作成と置換で、null の phoneNumbers と addresses を未割り当てとして受け付ける

## 動機

RFC 7643 §2.5 は、未割り当ての属性、null の値、空の配列を同じ状態とみなすことを SHALL で求める（2026-11-28 に本文を読んだ）。
IdMagic の SCIM の User の作成と置換は、`"phoneNumbers": null` と `"addresses": null` を「not a supported User attribute」として 400 の `invalidValue` で拒否する。
同じく対応しない `title` と `preferredLanguage` の null は受け付けており、扱いが揃っていない。
2026-11-28 に [[wi-21740-decide-recordings-for-requests-idmagic-receives]] で、Microsoft Learn の Entra ID の SCIM の手引き（Handling provisioning and deprovisioning of users の節）の POST の例を再生して見つけた。
この例は `addresses`、`phoneNumbers` などを null で送る。
グループの基準は p3 だが、規範に反してクライアントの作成を拒否する欠陥なので p1 とする。

## 対象範囲

- 作成と置換で、`phoneNumbers` と `addresses` の null と空の配列を、属性の省略と同じに扱う。null でない値の拒否は変えない。
- TypeSpec の `CreateScimUser` と `UpdateScimUser` の説明の「形式の誤った `phoneNumbers`、`addresses`」に、null と空の配列を含まないことを書く。
- [[wi-21740-decide-recordings-for-requests-idmagic-receives]] の `entraNotReplayed` から `create-user-with-nulls` を消し、記録の往復を再生する。

## 対象外

- `phoneNumbers` と `addresses` の値の保存。対応する属性を増やす別の判断である。
- PATCH の null の扱い。今回の観測では作成だけが拒否された。着手時に PATCH の `replace` の null も同じ規則に当たるかを確かめ、当たれば同じ項目で扱う。

## 設計

拒否の判断を、属性の名前の有無ではなく値の有無で行う。
null と空の配列を除いた後に、対応しない属性の値が残るときだけ拒否する。

## 計画

1. `spec-change` で TypeSpec の説明を直す。
2. 記録の再生で RED を観測し、実装を直す。

## タスク

- [ ] T001 [Spec] TypeSpec の説明を更新する。
- [ ] T002 [Acceptance] `create-user-with-nulls` の再生で RED を確認する。
- [ ] T003 [App] 作成と置換の検査を直す。
- [ ] T004 [Verify] 差分から選んだ検査を実行する。

## 検証

- `mise run test-go-package -- ./backend/sourcing/internal/scim/handlers_http`、`mise run test-go-package -- ./backend/sourcing/internal/scim/domain`。

## リスク

null でない不正な値まで受け付けると、送った値が黙って捨てられる。
null と空の配列だけを除き、値のある場合の拒否を検査で固定する。
