---
status: completed
---

# Directory Connector が送る差分から、外部の不変 ID で User を冪等に取り込む

Directory Connector が送る差分から、Connector と外部の不変 ID の組で User を取り込めるようにした（REQ-SOURCING-015〜019、状態機械 IngestionRunLifecycle）。
Connector API に取り込みの実行の開始、User のバッチ（100 件まで）、実行の完了を、管理 API に実行の一覧と詳細を加えた。
項目は一件ずつ IdManagement の作成、変更、無効化、削除の予約を通して確定し、作成は User と対応（`directory_user_correlations`）を同じトランザクションで確定する。actor は `directory_connector` である。
変わらない項目は User を書かず、同じバッチの再送は記録した結果を返し、内容の違う同じ ID は `batch_conflict` で拒否する。対応のない既存の User とユーザー名が衝突する項目は結び付けずに失敗として返す。
完全な再同期の完了は、報告した項目数が記録した数と一致するときだけ受け付け、実行に現れなかった同じ Connector の対応の `Active` の User だけを無効にする。新しい実行は同じ Connector の進行中の実行を打ち切る。
SCIM を拡張する案と、コンソール用の API を流用する案は採らなかった。前者は既存の SCIM の振る舞いを変え、サーバー側で完全な再同期を照合できず、後者は Connector に管理者の権限を与えるためである。
保守の負担を抑えるため、SCIM と Directory の属性の写し方を `sourcing/internal/sourceuser` に一つにまとめ、IdManagement の `SourceCommands` は actor を受け取り、拒否を公開するようにした。Directory の User は CSV のインポートが変更しない取り込み元の管理下の User に含めた（REQ-IDMANAGEMENT-004 への適合）。
差分の位置は Connector の手元だけに置き、`manager` の参照、削除予約中の User の復帰、衝突した項目の手動の結び付けは対象外とした。
変更耐性では、ドメインと `sourceuser` の変異をすべて検出し、ユースケース、保存先、組み立て地点への手の故障注入もすべて検出した。別のテナントの User への対応の検査は、対応がテナントで引かれるため本番から届かない防御である。失敗の一覧の上限 100 件の境界はテストで固定していない。
