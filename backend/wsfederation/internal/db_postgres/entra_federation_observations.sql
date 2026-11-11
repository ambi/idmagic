-- name: GetEntraFederationObservation :one
SELECT tenant_id, domain, observation, expected_fingerprint, signing_certificate_thumbprint,
 next_signing_certificate_thumbprint, matched, differences, retrieved_at, checked_at
FROM wsfed_entra_federation_observations WHERE tenant_id = $1 AND domain = $2;

-- name: ListEntraFederationObservations :many
SELECT tenant_id, domain, observation, expected_fingerprint, signing_certificate_thumbprint,
 next_signing_certificate_thumbprint, matched, differences, retrieved_at, checked_at
FROM wsfed_entra_federation_observations WHERE tenant_id = $1 ORDER BY domain;

-- 保存済みの記録より前に取得した観測では更新しない。影響行数 0 が古い観測を表す。
-- name: SaveEntraFederationObservation :execrows
INSERT INTO wsfed_entra_federation_observations (tenant_id, domain, observation, expected_fingerprint,
 signing_certificate_thumbprint, next_signing_certificate_thumbprint, matched, differences, retrieved_at, checked_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (tenant_id, domain) DO UPDATE SET observation=EXCLUDED.observation,
 expected_fingerprint=EXCLUDED.expected_fingerprint,
 signing_certificate_thumbprint=EXCLUDED.signing_certificate_thumbprint,
 next_signing_certificate_thumbprint=EXCLUDED.next_signing_certificate_thumbprint,
 matched=EXCLUDED.matched, differences=EXCLUDED.differences,
 retrieved_at=EXCLUDED.retrieved_at, checked_at=EXCLUDED.checked_at
WHERE wsfed_entra_federation_observations.retrieved_at <= EXCLUDED.retrieved_at;

-- name: DeleteEntraFederationObservation :exec
DELETE FROM wsfed_entra_federation_observations WHERE tenant_id = $1 AND domain = $2;
