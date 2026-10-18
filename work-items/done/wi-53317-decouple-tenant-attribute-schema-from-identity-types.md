---
status: completed
---

# テナント属性スキーマを IdManagement の保存型から切り離す

規範仕様の変更はない。
Tenancy の属性スキーマを管理用の値と要求側のポートにし、IdManagement の保存型と検証、組み込み属性、動的 Group の参照へ組み立て地点で接続した。
通知テンプレートの試し送りも、表示名と検証済みのメールだけを返す操作者照会に置き換えた。
保存先とテーブル所有者は IdManagement に維持し、User、Group、クレーム、CSV と worker の照会は IdManagement の保存ポートを使う。
Tenancy から IdManagement への本番 import を除き、消えた14件の負債だけを削除して、循環負債を45件から31件へ減らした。
利用者の指示に従って既存テストの入力と期待結果を保ち、fixture と参照を新しい境界へ追従させた。
ほかの8組の逆向きの辺と、属性の仕様、スキーマ更新の原子性、テーブルの移管は対象外とした。

全文は `git log -1 -- work-items/active/wi-53317-decouple-tenant-attribute-schema-from-identity-types.md` が示すコミットの親、またはそのコミットで読む。
