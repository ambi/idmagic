---
depends_on: []
status: pending
authors: [tn]
risk: medium
created_at: 2026-08-29
priority: p2
change_kind: bugfix
affected_spec:
  - { path: docs/contexts/provisioning/standards.md, requirement: RFC7643-OUT-CORE-RESOURCES }
  - { path: spec/contexts/provisioning/models.tsp, symbol: IdMagic.Contract.AttributeMappingRule }
---

# 外向き SCIM が送るリソース本文に `schemas` を入れる

## Motivation

下流へ送る User リソースの本文は、接続の属性対応付けが解決した属性だけで組み立てている。対応付けに `schemas` は無く、書こうとしても対象パスの文法が配列リテラルを取らない。結果として `POST /Users` の本文にも `PUT /Users/{id}` の本文にも `schemas` が現れない。

RFC 7643 §3 は、リソース表現が `schemas` を持つことを要求している。値は、そのリソースが従うスキーマの URN の集合であり、User なら `urn:ietf:params:scim:schemas:core:2.0:User` である。§8 の例はすべてこれを含む。

PATCH の本文だけは `urn:ietf:params:scim:api:messages:2.0:PatchOp` を持つので、欠けているのはリソース表現の側だけである。この非対称は、`schemas` を「メッセージの種別を言う欄」としてだけ扱い、リソース表現の必須属性としては扱っていないことを示している。

**これは連携先を選ぶ欠陥である。** 受け取り側が `schemas` の有無を検証する実装なら、作成も置換も 400 で拒否される。拒否は `RFC7644-OUT-ERROR-RESPONSE` が「再試行しない失敗」として扱う経路に落ちるため、配信は試行上限を待たずに死に、原因は下流のエラー本文にしか残らない。IdMagic 自身の内向きサーバーは `schemas` を読み取り専用属性として無視するので、IdMagic どうしを繋いだ試験では再現しない。

[docs/contexts/provisioning/standards.md](../docs/contexts/provisioning/standards.md) の `RFC7643-OUT-CORE-RESOURCES` を `partial` に留めているのはこの欠落のためである。直った時点で、この行の `Statement` は本文が `schemas` を持つことを言えるようになる。

## Scope

- 送出するリソース表現に `schemas` を入れる。User は core スキーマの URN を持つ。
- 属性対応付けが解決する属性とは別の層で入れるか、対応付けの既定に加えるかを決める。対応付けは管理者が編集できるので、必須属性を編集可能な表に置いてよいかがこの判断の要点になる。
- `RFC7643-OUT-CORE-RESOURCES` の `Statement` を更新し、証拠テストを足す。

## Out of Scope

- 拡張スキーマの URN を `schemas` に加えること。拡張属性は [[wi-403-provisioning-declares-no-scim-conformance]] が `RFC7643-OUT-SCHEMA-EXTENSIONS` として `excluded` を宣言しており、送らないものの URN を広告する理由が無い。
- Group リソースの送出。[[wi-441-push-groups-produces-no-delivery]] が持つ。
- 下流が返す `schemas` の検証。取り込みは行っていない。

## Design

未定。着手時に、必須属性を属性対応付けの外に置くか中に置くかを確定して本節に記録する。

## Plan

1. 送出本文の組み立てのどこに置くかを決める。
2. 作成と置換の本文が `schemas` を持つことのテストを RED で置く。
3. 実装し、`RFC7643-OUT-CORE-RESOURCES` の `Statement` を更新する。

## Tasks

- [ ] T001 [Design] 必須属性の置き場所を確定する。
- [ ] T002 [Test] 本文が `schemas` を持つことのテストを RED で置く。
- [ ] T003 [App] 実装する。
- [ ] T004 [Spec] `RFC7643-OUT-CORE-RESOURCES` の `Statement` を更新する。
- [ ] T005 [Verify] `mise run check-spec`、`mise run verify`。

## Verification

- `mise run check-spec`
- `mise run test-go`
- `mise run verify`

## Risk Notes

リスクは medium。送出する本文の形が変わるため、既に連携している下流の受け取り方が変わる。`schemas` を無視する実装では無害だが、未知の属性を拒否する実装では逆に壊れうる。既存の接続がある状態で送出形を変えることになるので、変更前後の本文を並べて確認する。
