-- name: CountRestoreBaseline :one
SELECT
  (SELECT count(*) FROM tenants) AS tenant_count,
  (SELECT count(*) FROM users) AS user_count,
  (SELECT count(*) FROM oauth2_clients) AS client_count;

-- name: ListTenantsMissingActiveSigningKey :many
SELECT t.id FROM tenants t
WHERE NOT EXISTS (
  SELECT 1 FROM signing_keys sk
  WHERE sk.tenant_id = t.id AND sk.key_usage = 'Signing' AND sk.active
)
ORDER BY t.id;

-- name: ListDuplicateActiveJobDedupKeys :many
-- 通常の運用では jobs_tenant_dedup_key_active_idx の一意索引がこの不変条件を守る。
-- 復元で索引が失われた場合や、行が索引を通らずに入った場合に備えて直接確かめ直す。
SELECT (tenant_id || ':' || dedup_key)::text AS tenant_dedup_key
FROM jobs
WHERE dedup_key IS NOT NULL AND status IN ('queued', 'running')
GROUP BY tenant_id, dedup_key
HAVING count(*) > 1
ORDER BY 1;

-- name: ListNonEmptyEphemeralTables :many
-- 復元時に空にする一時テーブルである。復元後に行が残っていれば、空にする手順が
-- 飛ばされたか失敗している。
SELECT t.table_name::text AS table_name
FROM (VALUES
  ('oauth2_authorization_requests', EXISTS (SELECT 1 FROM oauth2_authorization_requests)),
  ('oauth2_authorization_codes', EXISTS (SELECT 1 FROM oauth2_authorization_codes)),
  ('oauth2_par_requests', EXISTS (SELECT 1 FROM oauth2_par_requests)),
  ('oauth2_device_codes', EXISTS (SELECT 1 FROM oauth2_device_codes)),
  ('oauth2_replay_jtis', EXISTS (SELECT 1 FROM oauth2_replay_jtis)),
  ('oauth2_access_token_denylist', EXISTS (SELECT 1 FROM oauth2_access_token_denylist)),
  ('webauthn_sessions', EXISTS (SELECT 1 FROM webauthn_sessions)),
  ('login_throttle_counters', EXISTS (SELECT 1 FROM login_throttle_counters)),
  ('saml_authnrequest_replays', EXISTS (SELECT 1 FROM saml_authnrequest_replays))
) AS t(table_name, has_rows)
WHERE t.has_rows
ORDER BY t.table_name;
