# SigningKeys Decisions

- 鍵の一覧と参照は `AdminKeysRead` (`admin:keys_read`)、ローテーションは `TenantKeysRotate` (`admin:keys_rotate`)、検証用鍵の無効化は `TenantKeysDisable` (`admin:keys_disable`) を要し、いずれも `admin` または `system_admin` ロールを持つ、有効かつ認証済みのユーザーが所属テナントに対して行える。テナント横断の健全性一覧 `SystemKeyHealthRead` (`admin:keys_health_read`) だけは `system_admin` に限る。
- 管理 API のレスポンスに秘密鍵素材を含めることは、いずれの権限でもできない。返すのは `kid`、状態、有効期間、証明書、フィンガープリントに限る。公開鍵を配る JWKS とフェデレーションメタデータは認証を要さず、テナントごとのエンドポイントで公開する。
- 署名鍵は、差し替え可能な `KeyProvider` の背後でテナントごとに分離する。システム全体で鍵を共有したり、各プロトコルのアダプターにプロバイダーを埋め込んだりしない。
- 鍵のローテーション間隔（最短 90 日）、公開の重複期間（最短 7 日）、保管期間（7 年）は固定の規範的なポリシー値であり、文書化されない設定にはせずここに記録する。
