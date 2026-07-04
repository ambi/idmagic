-- idp-wi-114: rename application_sign_on_policies -> application_sign_in_policies
-- and migrate rule JSON to the constrained sign-in policy shape.
--
-- Run this explicit migration BEFORE applying deploy/schema/postgres.sql. psqldef
-- reads the declarative rename as DROP + CREATE (data loss), so the table rename
-- and JSON transform must be performed here first. After this script runs, the
-- psqldef dry-run against postgres.sql must be empty.
--
-- Idempotent: guarded by to_regclass so re-running is a no-op once renamed.

BEGIN;

DO $$
BEGIN
    IF to_regclass('public.application_sign_on_policies') IS NOT NULL
       AND to_regclass('public.application_sign_in_policies') IS NULL THEN
        ALTER TABLE application_sign_on_policies RENAME TO application_sign_in_policies;
    END IF;
END
$$;

-- Transform each rule to the constrained shape:
--   required_authn: { acr, factor } -> { strength }
--     strength = Mfa when acr = urn:idmagic:acr:mfa or factor = totp, else Password.
--   condition: drop free-text network / device (never evaluable); introduce
--     network_allow_cidrs (empty on migration); keep reauth_max_age_seconds.
UPDATE application_sign_in_policies AS p
SET rules = COALESCE(transformed.rules, '[]'::jsonb)
FROM (
    SELECT
        p2.tenant_id,
        p2.application_id,
        (
            SELECT jsonb_agg(
                jsonb_build_object(
                    'rule_id', rule->'rule_id',
                    'name', rule->'name',
                    'enabled', rule->'enabled',
                    'required_authn', jsonb_build_object(
                        'strength',
                        CASE
                            WHEN rule #>> '{required_authn,acr}' = 'urn:idmagic:acr:mfa'
                                 OR rule #>> '{required_authn,factor}' = 'totp'
                            THEN 'Mfa'
                            ELSE 'Password'
                        END
                    ),
                    'condition', (
                        CASE
                            WHEN (rule #> '{condition,reauth_max_age_seconds}') IS NOT NULL
                            THEN jsonb_build_object(
                                'reauth_max_age_seconds', rule #> '{condition,reauth_max_age_seconds}'
                            )
                            ELSE '{}'::jsonb
                        END
                    )
                )
            )
            FROM jsonb_array_elements(p2.rules) AS rule
        ) AS rules
    FROM application_sign_in_policies AS p2
    WHERE jsonb_typeof(p2.rules) = 'array' AND jsonb_array_length(p2.rules) > 0
) AS transformed
WHERE p.tenant_id = transformed.tenant_id
  AND p.application_id = transformed.application_id;

COMMIT;
