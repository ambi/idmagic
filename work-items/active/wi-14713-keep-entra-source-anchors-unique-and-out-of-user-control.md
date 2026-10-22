---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-23
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/ws-federation/features/relying-party.md, requirement: REQ-WSFEDERATION-001 }
  - { path: docs/modules/ws-federation/features/passive-sign-in.md, requirement: REQ-WSFEDERATION-002 }
  - { path: docs/modules/ws-federation/features/active-sts.md, requirement: REQ-WSFEDERATION-004 }
---

# Entra の sourceAnchor を、構成の後も一意で、利用者が変えられない値に保つ

## 動機

Entra は、Assertion の ImmutableID（sourceAnchor から導く値）で Entra のユーザーを決める。
同じ ImmutableID を二人の User に発行すると、片方がもう一方の Microsoft 365 のアカウントでサインインする。
sourceAnchor の検査には次の欠陥がある。

| 欠陥 | 結果 |
| --- | --- |
| 欠落、重複、変換できない値の検査は、Entra の構成の時点に一度だけ行う。その後に作ったか、属性を変えた User は検査しない | 構成の後に、管理者の操作、CSV、SCIM で別の User と同じ値を書けば、発行の時点で同じ ImmutableID を発行する |
| sourceAnchor の属性を `editable_by_user` にした属性スキーマを拒否しない | 利用者がセルフサービスのプロフィールで値を他人の値に書き換え、その人として Microsoft 365 にサインインできる |
| 構成の時点の検査は、テナントのすべての User を対象にする | Entra へフェデレーションしない User（ローカルの管理者や、ほかのドメインの User）が一人でも値を持たないと構成できない。初期データの User（`alice`）だけのテナントでも、`sourceAnchor attribute "object_guid" has no value` で拒否される |
| 拒否の理由は、英語の内部メッセージと User の ID をそのまま返し、管理画面がそれを表示する | 日本語の画面に英語の文が混ざり、どの User を直せばよいか、何件あるかが分からない |
| 汎用の RP の登録で、Entra の RP と同じ `wtrealm` を指定すると、定型設定と固定のクレームを黙って捨てて置き換える | フェイルクローズの定型設定を、手書きのクレームの設定で上書きできる |

## 対象範囲

- 発行の時点で、対象の User の ImmutableID がテナントのほかの User と重複していれば、発行を拒否する。
- sourceAnchor の属性が `editable_by_user` なら Entra の構成を拒否し、Entra が使っている属性を `editable_by_user` に変える属性スキーマの変更も拒否する。
- 構成の時点の検査の対象を、UPN がフェデレーションするドメインに属する User に絞る。
- 構成の拒否は、問題の種類（欠落、重複、変換できない値、利用者が編集できる属性）ごとの `code` と件数を返し、管理画面は辞書の文言で表示する。User の一覧は件数と、先頭の数件の表示名に限る。
- 汎用の RP の登録と削除で、Entra の定型設定を持つ RP を対象にした要求を拒否する。

## 対象外

- sourceAnchor の値の書き込みそのものを、Entra の構成の後に拒否すること。発行の時点で止めれば成り代わりは起きず、取り込みの経路（CSV、SCIM）ごとに Entra の規則を持ち込まずに済む。

## 設計

重複の判定を発行の時点に置くか、書き込みの時点に置くかを比べた。

| 案 | 採否 | 理由 |
| --- | --- | --- |
| 発行の時点に、同じ値の User が一人だけであることを確かめる | 採る | 書き込みの経路（管理 API、CSV、SCIM、セルフサービス）に依らず一箇所で守れる。発行の失敗は監査に残る |
| 属性の書き込みの時点に、一意性の制約を置く | 採らない | 属性は疎な対応表で保存しており、Entra が使う属性だけに一意索引を張るには、WsFederation の設定が IdManagement のスキーマを変えることになる |

発行の時点の判定には、sourceAnchor の属性の値で User を引く読み取りが要る。
IdManagement が公開する読み取りの操作を足すか、既存の検索で足りるかを T001 で確かめる。
足す場合はモジュール間の依存を変えるので、境界の判断の記録を書く。

## 計画

1. T001 で、属性の値による User の検索の手段を決める。
2. wi-97369 がドメインを複数にしたら、構成の時点の検査の対象を各ドメインに広げる。wi-97369 より先に実装するときは、現在の一つのドメインで絞る。

## タスク

- [ ] T001 [Plan] 属性の値による検索の手段を決め、要れば境界の判断の記録を書く。
- [ ] T002 [Spec] REQ-WSFEDERATION-001、002、004 と、属性スキーマの要件に拒否の条件を足す。
- [ ] T003 [Acceptance] 構成の後に重複させた User と、利用者が書き換えた sourceAnchor に発行することを RED で確かめる。
- [ ] T004 [App] 発行の拒否、属性スキーマの拒否、構成の検査の範囲、問題の `code` を実装する。
- [ ] T005 [UI] 構成の拒否を辞書の文言で表示し、件数と表示名を出す。
- [ ] T006 [Verify] 変更を検証する。

## 検証

- `mise run test-go-changed`、`mise run test-ui`、`mise run check`、`mise run check-spec-impact`。
- ミューテーションテスト：重複の判定と、`editable_by_user` の拒否。

## リスク

- 発行の時点の検索がサインインごとに走る。sourceAnchor の比較キーを索引で引けることを T001 で確かめる。
