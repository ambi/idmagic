---
status: completed
---

# IdManagement と Tenancy の操作が返すステータスとエラーコードを TypeSpec に宣言する

実装を変えず、実装と要件が返す応答を TypeSpec に宣言した。
IdManagement には PreviewNotReadyError と PreviewDigestMismatchError を加え、三種類の CSV のインポートの適用に 409 を、User と Group の適用に 404 を宣言した。
Tenancy には TenantConflictError、InvalidBrandingError、InvalidNotificationTemplateError を加え、テナントの作成に 409 を、ブランド設定の更新と通知テンプレートの保存とプレビューの 400 に固有のエラーを宣言した。400 の InvalidRequestError は、同じ操作が別の条件で返すので残した。
エクスポートの操作は、着手の時点ですでに宣言済みだった。
要件文は変えていない。
`check-status-drift` がエラー変換の関数を読まないので、宣言との一致は `//verify:operation` を付けた HTTP の契約テストで固定し、ステータスとコードを変える故障注入で各テストが失敗することを確かめた。
保存済みの公開前ベースラインとの互換性の診断は main と同一で、新しい違反はない。
表にない操作の漏れと、静的検査の解析の拡張は wi-12283 に残した。
