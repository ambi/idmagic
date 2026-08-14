# 環境別の seed プロファイル

seed はサーバーの起動時にはデフォルトで実行されない。次の形式で明示的にプレビューし、適用する。
`just seed <environment> <profile> <mode> [manifest]`

ローカルでの利便性のため、同じプロセス内で `development` プロファイルを明示的に適用するのは `just dev` と `just dev-memory` だけである。

プロファイルの望ましい状態は `seed/manifests/*.yaml` にある。4 番目の CLI 引数または環境変数 `SEED_MANIFEST` で別のルートマニフェストを選べる。マニフェストは厳密に解釈し、ローカルの相対パスによるインクルードだけを許可する。未知のキー、重複する論理キー、循環参照、ルート外のパス、YAML のアンカー・エイリアス・マージは、書き込みが起こる前に拒否する。

| Profile | Allowed environments | Contents |
| --- | --- | --- |
| `bootstrap` | development / test / staging / production | ファーストパーティークライアントのための最小限の設定。本番では `SEED_FIRST_PARTY_REDIRECT_URIS` が必須である。 |
| `development` | development / test / staging | ローカルのデモ用ユーザー、グループ、プロトコル例、アプリケーション。 |
| `test` | test | `development` と同一の決定的なフィクスチャー。 |
| `performance` | development / test / staging | 決定的に生成する合成ユーザー。通常は最大 10,000 件で、超える場合は `--allow-large` フラグが必要である。 |

`just seed development development dry_run` で変更をプレビューし、実行するときはモードを `apply` に変える。seed の出力には論理キーと件数だけを含め、パスワード、クライアントシークレット、TOTP seed、ハッシュ、個人識別情報の完全な値は含めない。

シークレットの値を YAML に直接置いてはならない。代わりに `provider`（`env` または `file`）、`locator`、`version` を使って参照する。ステージング環境と本番環境で許可するのは `file` プロバイダーだけで、ファイルの所在は `SEED_SECRET_ROOT` 配下の通常ファイルに厳しく制限する。`dry_run` モードでも参照を解決できるか検証する。

性能プロファイルを通常の検証フローに含めない。準備には `just seed development performance apply` で少ない件数を使い、計測時だけ `just seed-throughput development 10000 250` を実行する。10,000 件を超える場合は CLI フラグ `--allow-large` を指定し、100,000 を超える値は拒否する。
