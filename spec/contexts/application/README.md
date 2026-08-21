# Application

運用者が「接続する業務アプリケーション」として扱う Application を所有する。OIDC クライアント、SAML SP、WS-Fed RP は Application に関連付けるプロトコル設定である。表示名、アイコン、ライフサイクル、割り当て、サインインポリシー、ポータルでの並び順とカテゴリはここに集約する。

割り当てとサインインポリシーは、ポータルでの表示とフェデレーションの利用可否をフェイルクローズで制御する。通信時の動作は各プロトコルの Context が所有し、Application はプロトコル設定を中身に依存しないキーで参照する。

| File | Content |
|---|---|
| [glossary.md](glossary.md) | この Context での語義 |
| [decisions.md](decisions.md) | 設計判断 |
| [internals.md](internals.md) | 機構の説明 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |
