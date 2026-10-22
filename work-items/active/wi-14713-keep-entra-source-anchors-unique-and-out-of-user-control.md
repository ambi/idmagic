---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-23
priority: p2
depends_on: [wi-90369-keep-unique-user-attribute-values-unique-on-every-write]
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
| 欠落、重複、変換できない値の検査は、Entra の構成の時点に一度だけ行う。その後に作ったか、属性を変えた User は検査しない | 構成の後に、管理者の操作や CSV で別の User と同じ値を書けば、発行の時点で同じ ImmutableID を発行する |
| sourceAnchor は GUID（ハイフンの有無と大文字と小文字を問わない）と base64 のどちらの表記でも受け付ける | 同じ ImmutableID になる別の表記の値は、保存した値の一致では重複と分からない |
| sourceAnchor の属性を `editable_by_user` にした属性スキーマを拒否しない | 利用者がセルフサービスのプロフィールで値を他人の値に書き換え、その人として Microsoft 365 にサインインできる |
| 構成の時点の検査は、テナントのすべての User を対象にする | Entra へフェデレーションしない User（ローカルの管理者や、ほかのドメインの User）が一人でも値を持たないと構成できない。初期データの User（`alice`）だけのテナントでも、`sourceAnchor attribute "object_guid" has no value` で拒否される |
| 拒否の理由は、英語の内部メッセージと User の ID をそのまま返し、管理画面がそれを表示する | 日本語の画面に英語の文が混ざり、どの User を直せばよいか、何件あるかが分からない |
| 汎用の RP の登録で、Entra の RP と同じ `wtrealm` を指定すると、定型設定と固定のクレームを黙って捨てて置き換える | フェイルクローズの定型設定を、手書きのクレームの設定で上書きできる |

## 対象範囲

- sourceAnchor の属性は、wi-90369 の `unique` の属性に限る。構成の時点と発行の時点で、属性の定義が `unique` でないか `editable_by_user` なら拒否する。
- sourceAnchor の値を一つの表記に限り、ほかの表記を「変換できない値」として構成の時点と発行の時点で拒否する。表記は AD の objectGUID と Entra Connect の慣行を一次資料で確かめて T001 で決める。
- 構成の時点の検査の対象を、UPN がフェデレーションするドメインに属する User に絞る。
- 構成の拒否は、問題の種類（欠落、表記、属性の定義）ごとの `code` と件数を返し、管理画面は辞書の文言で表示する。User の一覧は件数と、先頭の数件の表示名に限る。
- 汎用の RP の登録で、Entra の定型設定を持つ RP を上書きする要求を拒否する。

## 対象外

- 値の重複の検査そのもの。wi-90369 が書き込みの時点で止める。
- 汎用の RP の削除の拒否。管理画面が Entra のフェデレーションを外す経路なので残す。

## 設計

重複を止める場所を比べ、書き込みの時点に置いた。

| 案 | 採否 | 理由 |
| --- | --- | --- |
| IdManagement が書き込みの時点で `unique` の属性の値の一意性を守る（wi-90369） | 採る | 値の持ち主が不変条件を守る。サインインの経路に処理を足さない |
| 発行の時点に、同じ値の User が一人だけであることを確かめる | 採らない | 属性に索引がなく、サインインのたびにテナントの User 数に比例する走査が入る。持ち主でないモジュールが不変条件を点検する |

一意性は保存した値の一致で判定するので、表記を一つに限ることで、同じ ImmutableID が別の値として保存される経路を塞ぐ。
発行の時点の検査は、対象の User 自身の値と属性の定義だけを見る。

## タスク

- [ ] T001 [Plan] sourceAnchor の表記を一次資料で確かめて決める。
- [ ] T002 [Spec] REQ-WSFEDERATION-001、002、004 に拒否の条件を足す。
- [ ] T003 [Acceptance] 一意でない属性、利用者が編集できる属性、別の表記の値に発行することを RED で確かめる。
- [ ] T004 [App] 属性の定義と表記の拒否、構成の検査の範囲、問題の `code`、RP の上書きの拒否を実装する。
- [ ] T005 [UI] 構成の拒否を辞書の文言で表示し、件数と表示名を出す。
- [ ] T006 [Verify] 変更を検証する。

## 検証

- `mise run test-go-changed -- main`、`mise run test-ui-unit-file`、`mise run check`、`mise run check-spec-impact`。
- ミューテーションテスト：属性の定義と表記の判定。

## リスク

- 表記を限ると、既存のテナントで別の表記の値を持つ User の発行が止まる。製品は未公開なので移行しないが、構成の時点の拒否で件数を示す。
