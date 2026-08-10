---
status: accepted
authors: [tn]
created_at: 2026-08-10
---

# ADR-162: SCIM の multi-valued emails/phoneNumbers/addresses は scim context 側の補助テーブルで持ち、`idmanagement.User` は単一 `Email` のまま変えない

## コンテキスト

wi-239([ADR-122](ADR-122-scim-mutation-atomicity-and-attribute-subset.md))は `RFC7643-CORE-RESOURCES`
を `adoption: partial` にし、`emails` を配列で受け取っても最初の要素の `value` だけを永続化していた。
wi-246 はこれを multi-valued(`emails` の type/primary 別複数値、`phoneNumbers`、`addresses`)まで広げる。
`idmanagement.User` は単一 `Email *string` フィールドしか持たず、`Attributes map[string]AttributeValue`
(ADR-039/040)は String/Number/Boolean/Date/StringArray のスカラー sum type のみを表現でき、`emails[].type`
+ `primary` や `addresses` のような「複数要素それぞれが複数の sub-attribute を持つ」構造を表現できない。
`idmanagement.User` のモデル変更が必要かを実装前に判断する必要があった([wi-246](../work-items/wi-246-scim-multivalued-core-attributes-and-nested-group-members.md) Scope)。

## 決定

**`idmanagement.User` は変更しない。** multi-valued emails/phoneNumbers/addresses は `sourcing/scim`
bounded context 側の補助テーブルに保持し、そのうち primary email だけを従来どおり `idmanagement.User.Email`
に同期する。設計の詳細(テーブル形状、同期タイミング)は実装着手時に `backend/sourcing/scim/ARCHITECTURE.md`
に書く。

## 却下した代替案

- **`idmanagement.User` に `Emails []Email` / `PhoneNumbers []Phone` / `Addresses []Address` を追加する**:
  `Attributes` の sum type はスカラー値しか持てず、複数 sub-attribute を持つ複数要素という新しいモデリング
  プリミティブが要る。認証・通知テンプレート・管理コンソール UI・CSV export/import・data export
  (`backend/idmanagement/domain/data_export.go`)など、`User.Email` を単一値として前提する全消費者に波及し、
  SCIM resource contract という本 WI のスコープを大きく超える(ADR-122 と同じ判断軸)。
- **`Attributes` bag に `emails`/`phone_numbers`/`addresses` を StringArray として押し込む**: `type`/`primary`
  のような per-要素メタデータを表現できず、round-trip で情報が失われる。RFC7643 準拠を謳う以上不可。

## 影響

- `spec/contexts/scim.yaml` の `RFC7643-CORE-RESOURCES` requirement(実装時に更新)。
- 新規 SCL 要素・補助テーブルは `sourcing/scim` context に閉じる。`idmanagement` 側の interface/model は不変。
- 実装時、`backend/sourcing/scim/ARCHITECTURE.md` にテーブル形状と同期方針を記述する。
