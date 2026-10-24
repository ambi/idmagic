# SignInPolicy のデータ

## 集約

| 集約 | 内包する主な値 | 他の集約への参照 |
| --- | --- | --- |
| `AppSignInPolicy` | 順序付けた `SignInRule` の集合 | Application の識別子とテナントを参照する |
| `TenantDefaultSignInPolicy` | テナントのデフォルトの `SignInRule` の集合 | Tenant の識別子を参照する |

ポリシーの条件と保存する規則の型は、このモジュールが所有する。
既存の HTTP 形式と TypeSpec のモデル名は Application の公開契約として維持する。

## 保存と削除

メモリと PostgreSQL の保存アダプターは、同じ公開の保存ポートを満たす。
PostgreSQL は既存のポリシー表を使い、規則の JSON、アプリとテナントへの外部キー、および削除時の cascade を維持する。
アプリの実在は Application の照会から得て、更新先は SignInPolicy が所有するポリシー表だけにする。
テナントで絞る SQL は、従来どおり Application の行との結合で確認する。

Application の通常の削除は、割り当て、ポリシー、アプリの順で削除し、その後に使用量を減らしてイベントを発行する。
ポリシーの削除を所有者のポート越しに呼ぶことで、この順序を維持する。
複数の保存先の削除を一つのトランザクションに変えず、元の削除が持たない原子性を追加しない。
Application か Tenant の親行が消えるときの cascade も残す。
