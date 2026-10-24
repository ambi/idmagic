---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-24
priority: p2
depends_on: [wi-10848-keep-saml-and-wsfed-configuration-updates-with-their-owners]
change_kind: bugfix
affected_spec:
  - { path: docs/modules/application/features/catalog.md, requirement: REQ-APPLICATION-007 }
  - { path: docs/modules/application/features/catalog.md, requirement: REQ-APPLICATION-005 }
---

# Application の作成と更新で、SAML の SP と WS-Fed の RP を所有者と同じ規則で拒否する

## 動機

[[wi-10848-keep-saml-and-wsfed-configuration-updates-with-their-owners]] の照合で、ApplicationCatalog の経路が SP と RP の管理 API と異なる結果を返すことを確認した。
構造変更の中では揃えず、この項目で仕様を定めて直す。

| 経路 | 現在の結果 | 所有者の管理 API の結果 |
| --- | --- | --- |
| `wsfed` と `saml` の Application の作成 | 同じ `wtrealm` または entityID の RP と SP を確かめずに置き換える。Entra の RP（`urn:federation:MicrosoftOnline`）なら `entra_profile` が消え、別の Application に属する RP と SP は設定だけが置き換わる。PostgreSQL の linker は未結合の行だけを結ぶので、新しい Application は他人の RP と SP を指す | Entra の RP と同じ `wtrealm` の登録を 409 と `entra_relying_party` で拒否する |
| 作成とプロトコル設定の更新 | `Validate()` を呼ばず、2,048 文字または 2,048 バイトを超える `wtrealm` と entityID を拒否しない | 422 と `field_length_exceeded` で拒否する |
| プロトコル設定の更新のイベント | クレームを変えたときだけ `ApplicationClaimMappingUpdated` を発行する | REQ-APPLICATION-005 は `ApplicationUpdated` の発行を定める |

## 対象範囲

- 既存の RP と SP と同じキーの Application の作成を拒否する状態コードとエラーを、REQ-APPLICATION-007 に定める。Entra の RP と、Application に属する RP と SP を含める。
- 作成と更新で、長さの上限を所有者と同じ 422 と `field_length_exceeded` で拒否する。
- REQ-APPLICATION-005 のイベントの記述と実装の不一致を、どちらに合わせるか決めて揃える。
- 拒否は wi-10848 で定めた `RelyingPartyAdministration` と `ServiceProviderAdministration` の中で行い、Catalog はエラーを HTTP へ写すだけにする。

## 対象外

- 所有者の管理 API の upsert の意味の変更。
- 作成の途中で失敗したときに残る RP と SP の補償。

## 設計

未結合の既存の RP と SP を Application の作成で引き取るか拒否するかを、仕様の変更の前に決める。
拒否する場合は 409 を候補とし、所有者の管理 API の `entra_relying_party` と `application_owned_protocol` のコードとの関係を整理する。

## 計画

1. REQ-APPLICATION-007 と REQ-APPLICATION-005 を更新し、`mise run check-spec` を通す。
2. Catalog の HTTP の境界で RED を観測してから、所有者の公開操作で拒否する。
3. メモリーと PostgreSQL の両永続層で確認する。

## タスク

- [ ] T001 [Spec] 作成時の重複、長さの上限、更新のイベントを仕様に定める。
- [ ] T002 [Acceptance] Catalog の HTTP の境界で各拒否の RED を確認する。
- [ ] T003 [App] 所有者の公開操作で拒否し、Catalog でエラーを写す。
- [ ] T004 [Verify] 両永続層と契約テストで確認する。

## 検証

- `mise run test-go-changed`：Catalog と両所有者を race 検査する。
- `mise run check-contract-drift` と契約テスト：追加する拒否の応答を契約と照合する。
- `mise run check-spec`：規範の変更を検査する。

## リスク

既存の未結合の RP と SP を拒否すると、所有者の管理 API で先に登録してから Application を作る手順が使えなくなる。
仕様の決定の前に、その手順の利用の有無を確かめる。
