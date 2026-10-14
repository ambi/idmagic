---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# エンドユーザー向け「マイページ」(Account Portal) を実運用相当で用意する

end-user 向け account portal (`/account`) を admin shell とは別の AccountShell で
実装し、以下のステージを段階的にリリースした:
- shell + アカウント概要 (`/api/account/summary`, roles を含まない self 契約)
- 個人情報 (表示名・editable 属性、wi-19 の self 経路)
- primary email 変更 (ワンタイムリンク再検証、ADR-030 方式)
- セキュリティ: パスワード変更導線 + TOTP の self-service 登録・解除
(登録時 QR コード [[wi-40-totp-enrollment-qr-code]]、所持証明で保護)
- アクティビティ: サインイン履歴 + 有効なセッションの一覧・個別失効・
他端末ログアウト (バックエンドは [[wi-20-authentication-event-history]]
スライス 1/2)
- 接続済みアプリ: consent の一覧と revoke
- データとプライバシー: 個人データの同期 JSON エクスポート
全 self エンドポイントは認証セッションの sub / tenant_id に固定し、mutation は
CSRF + same-origin で保護した。
残件は独立 WI に分離した: secondary / recovery 連絡先 →
[[wi-41-secondary-and-recovery-email]] / WebAuthn・passkey・recovery codes →
[[wi-26-webauthn-passkey-and-recovery-codes]] / 高 sensitivity 操作の横断 step-up
再認証 (ADR-043) → [[wi-43-account-portal-step-up-auth]]。data export の非同期ジョブ / object
storage 連携は引き続き out_of_scope (同期生成で提供)。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
