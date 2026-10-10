# SCIM による取り込みの設計

この文書は、[SCIM による取り込み](README.md)の要件を保証する仕組みを扱う。

## アーキテクチャ

User の変更では、すべてのメールアドレスの要素を検証した後、`primary=true`、大文字と小文字を区別しない `type=work`、ワイヤの上の先頭という優先順位で 1 件へ投影する。

`manager` は SCIM の ID への参照として受け取り、`ScimUserRef` を介して同じテナントの `User.sub` へ解決する。
レスポンスの `schemas` には、いずれかの Enterprise 拡張の属性を持つ場合だけ Enterprise 拡張の URN を含める。

`active` を `false` にする変更と、`DELETE /Users/{id}` による削除の予約は、User を直接保存せず、`UserLifecycle` のポートを通して IdManagement の User の操作で行う。
管理 API で止めたときと同じイベント、記憶済みの端末の失効、下流への通知、所有する Agent の無効化が伴う。
それ以外の User と Group の作成と更新、Group のメンバーシップは、IdManagement の Repository を直接保存する（[Sourcing のリスク](../design/risks.md)）。

`DELETE /Groups/{id}` は即時かつ完全である。
Group は個人識別情報を持たないからである。

## 信頼性

Group の作成で、対応の記録やメンバーの追加に失敗した場合は、作った Group と対応の記録を消してからエラーを返す。
この後始末の失敗は捨てるので、Group だけが残ることがある。
残った Group は、外部の IdP から見えない孤立した Group になるので、管理 API で消す。
