---
status: completed
archived_from: 3ae2ae69f793e14a35138e05bbaa367b50666b2e
---

# フロントエンドの重要操作を単体テストで保護し、カバレッジを引き上げる

`frontend/src/api/` の未テスト API クライアント（account、admin、authFlow、oidc）にリクエスト内容・成功応答・失敗応答のテストを追加済みだった状態から、後発の `webauthn.ts`（パスキーの JSON ⇔ ブラウザ API 変換）に対して `webauthn.test.ts` を新規追加し、Scope で挙げた全 API クライアントのテストを揃えた。

全文は `git show 3ae2ae69f793e14a35138e05bbaa367b50666b2e:<このファイルのパス>` で読む。
