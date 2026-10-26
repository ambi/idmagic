-- スキーマ適用後、新しい API と worker の起動前に実行する。
-- 既存の所属とロールは保持し、自動承認は作らない。再実行しても結果は同じである。
-- 後退で印を外すと上流による管理権限の付与を再び許すため、印を自動で戻さない。
-- 旧コードへ戻す前に対象のグループからロールを外し、必要に応じて所属の同期も停止する。
UPDATE groups AS g
SET upstream_managed = TRUE
WHERE NOT g.upstream_managed
  AND EXISTS (
    SELECT 1 FROM scim_group_refs AS ref
    WHERE ref.tenant_id = g.tenant_id AND ref.group_id = g.id
  );
