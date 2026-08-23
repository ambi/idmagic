# Authorization

リソース 1 件ごとの細粒度な認可を担う。テナントごとの認可モデル (リソース型と関係の定義) と関係タプル (`resource ⇄ subject` の事実) を持ち、それらをたどって「この主体はこのリソースに対してこの関係を持つか」を判定する。粗いロールでは表現できない「ユーザー U は文書 D を読めるか」に答えるための Context である。

最終的な認可判断の合成は担わない。この Context は関係が成立するかという**事実**を求め、OAuth2 側の AuthZEN スタイルの `Authorizer` ポートへ渡す。ロール、スコープ、代行チェーン、プリンシパルの有効性との論理積は、評価器側の規則表が担う。これにより、外部 PDP へ差し替えても合成規則が重複しない。

管理 API を誰が呼べるかというロール認可は、引き続き OAuth2 の規則表が担う。本 Context が扱うのは、データ資源へのアクセス判定である。

| File | Content |
|---|---|
| [glossary.md](glossary.md) | この Context での語義 |
| [standards.md](standards.md) | 準拠する外部規範 |
| [decisions.md](decisions.md) | 設計判断 |
| [internals.md](internals.md) | 機構の説明 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |
