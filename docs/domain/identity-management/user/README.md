# ユーザー

`User` の作成、管理者による一覧、無効化と再有効化、削除の予約と復元、完全削除、プロフィール属性を扱う。
コードの機能スライスは `backend/idmanagement/user` である。
本人によるセルフサービスの操作は[アカウントのセルフサービス](../account/README.md)が、CSV による一括の作成と更新は[ユーザー CSV](../user-csv/README.md)が扱う。

| 文書 | 内容 |
|---|---|
| [ユーザーの状態遷移](states.md) | 状態と遷移 |
| [ユーザーの設計判断](decisions.md) | 設計判断 |
| [ユーザーの内部設計](internals.md) | 機構の説明 |
| [ユーザーのシナリオ](scenarios.feature.md) | 受け入れシナリオ |
