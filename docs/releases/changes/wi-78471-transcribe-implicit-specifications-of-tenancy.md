# wi-78471-transcribe-implicit-specifications-of-tenancy

Tenancy がこれまで実装だけで守っていた挙動を、規則として文書で約束するようになった。
振る舞いは変わらない。
利用者が依存してよい境界値、デフォルト値、応答の形が、次の規則で読めるようになった。

- テナントの解決：Host の照合の規則、拒否の応答の形、`Vary: Host`、発行者の導出（[`REQ-TENANCY-022`](../../modules/tenancy/resolution/README.md)〜`REQ-TENANCY-024`）
- テナントのライフサイクル：realm に使える文字と予約語、一覧の順序、無効化と再開の再実行（[`REQ-TENANCY-025`](../../modules/tenancy/lifecycle/README.md)〜`REQ-TENANCY-027`）
- テナント設定：信頼済みデバイスの有効期間の上限、通知のデフォルト言語、パスワードポリシーの上書きの正規化、`TenantUpdated` の `changed_fields`（[`REQ-TENANCY-028`](../../modules/tenancy/settings/README.md)〜`REQ-TENANCY-031`）
- ブランド設定：文字列の上限と正規化、画像の形式と 256 KiB の上限、公開取得の ETag、画像配信のヘッダー（[`REQ-TENANCY-032`](../../modules/tenancy/branding/README.md)〜`REQ-TENANCY-035`）
- リソース上限：Hard Quota のデフォルト値と実効値、上限の更新が上書きの全体を置き換えること（[`REQ-TENANCY-036`](../../modules/tenancy/quota/README.md)、`REQ-TENANCY-037`）
- 通知テンプレート：プレビューの補完、リセットのイベント、試し送りの言語と応答（[`REQ-TENANCY-038`](../../modules/tenancy/notification-template/README.md)〜`REQ-TENANCY-040`）
- 連携エンドポイント：署名証明書のフィンガープリントの形式と、資格情報を解決できないときの 503（[`REQ-TENANCY-041`](../../modules/tenancy/integration-endpoints/README.md)、`REQ-TENANCY-042`）
- 属性スキーマ：動的グループが参照するユーザー属性の削除と型の変更の拒否（[`REQ-TENANCY-043`](../../modules/tenancy/attribute-schema/README.md)）

いくつかの規則には、維持するか是正するかが決まっていない点を要判断として残した。
是正する場合は、別の変更で知らせる。
