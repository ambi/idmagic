import { defineDictionary } from '../../lib/i18n'

export const systemDataKeyHealthDictionary = defineDictionary(
  {
    fetchFailedError: 'テナント別の暗号鍵の状態を取得できませんでした。',
    pageTitle: '暗号鍵の状態（全テナント）',
    pageDescription:
      '可逆な秘密情報（MFA の TOTP シードなど）を保護するテナント別データ暗号鍵 (DEK) の状態と、鍵管理プロバイダの到達性を横断で確認します。',
    reloadAriaLabel: '一覧を再読み込み',
    tableHeaderTenant: 'テナント',
    tableHeaderActiveVersion: '現在のバージョン',
    tableHeaderStatus: '状態',
    tableHeaderProvider: 'プロバイダ',
    tableHeaderProviderStatus: 'プロバイダ状態',
    tableHeaderRotatedAt: '最終ローテーション',
    healthy: '正常',
    unreachable: '接続不可',
    noTenantsNotice: '暗号鍵が発行されたテナントがありません。',
    statusActive: 'active',
    statusRetiring: 'retiring',
    statusDisabled: 'disabled',
    statusDestroyed: 'destroyed',
  },
  {
    fetchFailedError: 'Could not fetch per-tenant data key health.',
    pageTitle: 'Data key health (all tenants)',
    pageDescription:
      "Check each tenant's data encryption key (DEK) status and master-key provider reachability across the fleet — the keys protecting reversible secrets such as MFA TOTP seeds.",
    reloadAriaLabel: 'Reload the list',
    tableHeaderTenant: 'Tenant',
    tableHeaderActiveVersion: 'Active version',
    tableHeaderStatus: 'Status',
    tableHeaderProvider: 'Provider',
    tableHeaderProviderStatus: 'Provider status',
    tableHeaderRotatedAt: 'Last rotated',
    healthy: 'Healthy',
    unreachable: 'Unreachable',
    noTenantsNotice: 'No tenant has a data key yet.',
    statusActive: 'active',
    statusRetiring: 'retiring',
    statusDisabled: 'disabled',
    statusDestroyed: 'destroyed',
  },
)
