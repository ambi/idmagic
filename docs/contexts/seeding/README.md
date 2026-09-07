# Seeding

環境ごとの初期データを、望ましい状態として宣言し、計画し、適用する責務を担う。マニフェストの文法、環境ごとに許すプロファイルとシークレット提供元、プレビューと適用の計画器、シークレットを伏せた計画の表現がここに属する。

業務データそのものは扱わない。Tenant、User、Group、Application の記録と永続化は、それぞれの Context に残る。この Context は各 Context が公開するコマンドを呼ぶだけで、SQL フィクスチャで行を直接書き込む経路を持たない。

HTTP の接点も持たない。seed を実行できるかどうかは権限ではなく、プロセスを起動できる実行環境そのものが境界である。

| 文書 | 内容 |
|---|---|
| [Seeding の用語集](glossary.md) | この Context での語義 |
| [Seeding の設計判断](decisions.md) | 設計判断 |
| [Seeding の内部設計](internals.md) | 機構の説明 |
| [Seeding Scenarios](scenarios.feature.md) | 受け入れシナリオ |
