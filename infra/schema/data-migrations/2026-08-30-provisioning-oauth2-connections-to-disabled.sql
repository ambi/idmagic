-- wi-440 / auth_method='oauth2_client_credentials' の接続が、クライアント資格情報
-- フローで実際にトークンを取得するようになった。それまでは保存した client_secret を
-- そのままベアラートークンとして送っており、この方式の接続は 1 件も配信できていない。
--
-- トークン取得に要る token URL / client_id / scope は、以前は API が受理して検証した
-- あと捨てられていた。したがって既存の行には値が無く、埋め直す元データも存在しない。
-- 管理者が設定し直す以外に有効化する道は無い。
--
-- 更新後の infra/schema/postgres.sql を適用する前に 1 度実行する。新しい
-- provisioning_connections_oauth2_credential_check は、token URL と client_id が
-- 空のまま active な oauth2 接続を拒否する。
--
-- 冪等: 1 度目の実行後に条件へ合致する行は残らない。一方向の移行であり
-- (disabled にするだけ)、再び active になるのは管理者が資格情報を設定し直して
-- 明示的に有効化したときだけである。自動では戻さない。
UPDATE provisioning_connections
SET status = 'disabled', updated_at = now()
WHERE auth_method = 'oauth2_client_credentials'
  AND status = 'active'
  AND (credential_oauth2_token_url IS NULL OR credential_oauth2_token_url = ''
       OR credential_oauth2_client_id IS NULL OR credential_oauth2_client_id = '');
