---
status: completed
---

# IdManagement の OAuth2 操作と照会を要求側のポートで分離する

`spec-diff -- main` に規範仕様の変更はない。
IdManagement が求める削除とクライアントの存在照会へ契約を絞り、既存の保存先を直接渡す。
本人データの同意は既存の応答型を要求側の表示値へ移し、組み立て地点で一度だけ変換する。
User の削除順、途中失敗からの再実行、Agent のテナント境界と束縛の判断、HTTP の項目とイベントは保つ。
新しい中継層とパッケージを増やさず、IdManagement から OAuth2 への本番 import を除き、観測されなくなった 13 件の循環を台帳から削除した。
テストを変更しない構造変更なので、既存テストと配置の検査を使い、RED と変更耐性の検査は適用しない。
使用量の再集計、削除の原子性、Agent の資格情報とエクスポート項目の変更は対象外に残す。

全文は `git log -1 -- work-items/active/wi-62674-isolate-identity-oauth-lifecycle-and-account-queries.md` が示すコミットの親、またはそのコミットで読む。
