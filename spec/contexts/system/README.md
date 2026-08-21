# System

システムの入口を担う。ブラウザーが最初に触れる面 — ホステッドの認証画面 (ログイン、同意、デバイス認証)、管理コンソール、アカウントポータル — と、それらを支える認可トランザクション、API の経路の分け方、表示言語の解決がここに属する。

業務データそのものは扱わない。どのユーザーに何ができるかは記録の正を持つ各 Context が決め、この Context が決めるのは、どの経路へどの資格情報で到達できるかである。

製品全体が従う外部規範は [spec/standards.md](../../standards.md)、Context を跨ぐ語は [spec/glossary.md](../../glossary.md)、実行単位と信頼境界は [spec/deployment.md](../../deployment.md) が持つ。実行手順と検証コマンドは仕様ではなく、リポジトリの `README.md` にある。

| File | Content |
|---|---|
| [glossary.md](glossary.md) | この Context での語義 |
| [decisions.md](decisions.md) | 設計判断 |
| [internals.md](internals.md) | 機構の説明 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |
