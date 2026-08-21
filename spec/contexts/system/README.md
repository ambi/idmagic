# System

外部標準、共有語彙、横断的なユーザー体験、複数の Context にまたがるシナリオを所有する。

React UI と Go API は別々にビルドし、ゲートウェイを通じて同一オリジンで公開する。組み込みの認証画面 (ログイン、同意、デバイス認証)、管理コンソール、アカウントポータルはこの Context に属する。

実行手順と検証コマンドは所有せず、`README.md` に置く。

| File | Content |
|---|---|
| [glossary.md](glossary.md) | この Context での語義 |
| [standards.md](standards.md) | 準拠する外部規範 |
| [decisions.md](decisions.md) | 設計判断 |
| [internals.md](internals.md) | 機構の説明 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |
