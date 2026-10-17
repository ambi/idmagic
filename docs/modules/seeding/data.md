# Seeding のデータ

この文書は、Seeding の Aggregate を扱う。

## 集約

このモジュールは Aggregate を定義しない。
`SeedManifest` と `SeedPlan` は 1 回の実行の中でしか生きない値であり、投入した業務データの記録の正は各モジュールに残る。

- **判断**：状態を保存しない理由は、[Aggregate を定義せず投入の結果を保存しない](decisions.md#aggregate-を定義せず投入の結果を保存しない)。

| 値 | 内容 |
| --- | --- |
| `SeedRequest` | 環境、プロファイル、`dry_run` か `apply` か、マニフェストのパス、ファーストパーティーのリダイレクト URI、生成器の seed、件数、バッチの大きさ |
| `SeedManifest` | seed のリソースと決定的な生成器を宣言する、バージョン付きの YAML |
| `SeedSecretReference` | マニフェストがシークレットの値の代わりに持つ、提供元、ロケーター、バージョンの組 |
| `SeedPlan` | 現在の状態とマニフェストを比べて作る、論理キーごとの `create`、`update`、`noop`、`conflict` の操作の列。シークレットを含まない |

論理キーと ID は、プロファイルと生成器の seed から決定的に導く。
同じ要求を再実行すると、同じ論理キーを同じ ID で計画する。
