# System

システムの入口を担う。ブラウザーが最初に触れる面 — ホステッドの認証画面 (ログイン、同意、デバイス認証)、管理コンソール、アカウントポータル — と、それらを支える認可トランザクション、API の経路の分け方、表示言語の解決がここに属する。

業務データそのものは扱わない。どのユーザーに何ができるかは記録の正を持つ各 Context が決め、この Context が決めるのは、どの経路へどの資格情報で到達できるかである。

プロダクト全体が従う外部規範は [全体規範](../../standards.md)、Context を跨ぐ語は [用語集](../../glossary.md)、実行単位は [実行時アーキテクチャ](../../architecture/runtime.md)、信頼境界は [脅威モデル](../../design/security/threat-model.md) が持つ。実行手順と検証コマンドは仕様ではなく、リポジトリの `README.md` にある。

| 文書 | 内容 |
|---|---|
| [System の用語集](glossary.md) | この Context での語義 |
| [System の設計判断](decisions.md) | 設計判断 |
| [System の内部設計](internals.md) | 機構の説明 |
| [System Scenarios](scenarios.feature.md) | 受け入れシナリオ |
