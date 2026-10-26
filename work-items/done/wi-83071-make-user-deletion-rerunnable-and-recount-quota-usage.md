---
status: completed
---

# テナントの User の使用量を、実際の件数から再集計して補正する

REQ-TENANCY-049 を追加し、保持期限バッチ（`retention-sweep`）が完全削除の後に、各テナントの User の使用量を使用量を占める User の件数で置き換えるようにした。
数えるのは完全削除されていない User と、使用量の減算を終えていない（`pending_purge` が `release_usage` の）Tombstone である。
Tenancy の `QuotaRepository.RecountUsage` は使用量の行を `FOR UPDATE` で固定したまま件数を数えて置き換えるので、数えている間の加算と減算は置き換えの後に適用される。件数は IdManagement の `UserRepository.CountHoldingQuota` が自分の接続で数え、モジュールをまたぐトランザクションは作らない。
置き換え前の使用量と件数が異なれば警告ログを残し、イベントは発行しない。測定または保存に失敗したテナントは使用量を保ち、残りを処理してからバッチの失敗を返す。
作成の加算と User の保存の間に数えた User による一時的なずれは残し、次の再集計で直る。ユーザーの設計、IdManagement のリスク、Tenancy の README、実行時構成とデータベースの設計から、使用量を手で直す手順を再集計による回収へ改めた。
