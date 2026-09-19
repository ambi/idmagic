# Application

運用者が「接続する業務アプリケーション」として扱う Application は、この Context に属する。OIDC クライアント、SAML SP、WS-Fed RP は Application に関連付けるプロトコル設定である。表示名、アイコン、ライフサイクル、割り当て、サインインポリシー、ポータルでの並び順とカテゴリはここに集約する。

割り当てとサインインポリシーは、ポータルでの表示とフェデレーションの利用可否をフェイルクローズで制御する。通信時の動作は各プロトコルの Context が担い、Application はプロトコル設定を中身に依存しないキーで参照する。

| 文書 | 内容 |
|---|---|
| [Application の用語集](glossary.md) | この Context での語義 |
| [Application の設計判断](decisions.md) | 設計判断 |
| [Application の内部設計](internals.md) | 機構の説明 |
| [Application のシナリオ](scenarios.feature.md) | 受け入れシナリオ |
