# Seeding Internals

## SeedData
seed を同じ決定的な計画器でプレビューし、適用する内部運用インターフェースである。適用時は記録を所有する各 Context が公開するコマンドだけを呼び、SQL フィクスチャを直接使って不変条件を迂回しない。
- Input invariant: manifest_schema_supported(input.request)
- Input invariant: manifest_profile_matches_request(input.request)
- Input invariant: manifest_paths_are_local_and_contained(input.request)
- Input invariant: input.request.environment in ['staging', 'production'] implies manifest_secret_providers(input.request) == ['file']
- Result invariant: input.request.mode == 'dry_run' implies persistent_state_unchanged()
- Result invariant: reapply_same_request_is_noop(input.request)
- Result invariant: input.request.environment == 'production' && input.request.profile == 'bootstrap' implies production_safe_redirect_uris(input.request.first_party_redirect_uris)
- Result invariant: seed_plan_and_diagnostics_exclude_secret_values(output.plan)

## Environment policy and planning

本番が受け付けるのは `bootstrap` プロファイルだけであり、それ以外は書き込み前にフェイルクローズで拒否する。これにより、宛先を誤ったリクエストからデモ用の資格情報が本番へ投入されることを防ぐ。

適用は Context をまたぐ単一トランザクションではなく、依存順に並べた上限付きバッチの冪等なコマンドで行う。`performance` プロファイルのバッチサイズはデフォルト 250、上限 1,000 とする。専用の進捗テーブルは設けない。論理キーと ID をプロファイルと生成 seed から決定的に導くため、途中で失敗しても同じリクエストを再実行すれば未完了分だけが適用される。直列化には、リクエストキーごとのプロセス内ミューテックスと、プロセス間では既存の接続に対する PostgreSQL アドバイザリーロックを使う。

## Seed manifests and secret references

`models.SeedManifest` は、バージョンを持ち厳密に解釈する YAML の望ましい状態である。`manifests_yaml` アダプターが Domain の型へ変換してから、リソースごとの既存処理へ渡す。Domain とユースケースは、パーサー、ファイルシステム、環境変数 API を直接参照しない。`include` で読み込めるのはマニフェストルート配下のローカル相対パスだけとし、深さと総数に上限を設ける。パストラバーサルや注入の余地を避けるため、YAML のマージキー、テンプレート展開、リモート URL、環境変数展開は文法から除外する。シークレット値はマニフェストに直接書かず、`models.SeedSecretReference` で参照する。`env` プロバイダーはどの環境でも使えるが、ステージングと本番で許可するのは `file` プロバイダーだけである。プレビューでは参照を解決できることを検証するが、取得した値を計画、ログ、エラーへ渡すことは一切ない。
