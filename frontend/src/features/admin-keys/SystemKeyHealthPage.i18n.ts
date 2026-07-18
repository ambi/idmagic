import { defineDictionary } from '../../lib/i18n'

export const systemKeyHealthDictionary = defineDictionary(
  {
    fetchFailedError: 'テナント別の署名鍵の状態を取得できませんでした。',
    pageTitle: '署名鍵の状態（全テナント）',
    pageDescription:
      '全テナントの署名鍵プロバイダ（Local / DB / VaultTransit）の稼働状況と現在の署名鍵 ID を横断で確認します。',
    reloadAriaLabel: '一覧を再読み込み',
    tableHeaderTenant: 'テナント',
    tableHeaderProvider: 'プロバイダ',
    tableHeaderActiveKid: '現在の署名鍵 ID',
    tableHeaderKeyCount: 'JWKS 鍵数',
    tableHeaderProviderStatus: 'プロバイダ状態',
    healthy: '正常',
    unreachable: '接続不可',
    noTenantsNotice: 'テナントがありません。',
  },
  {
    fetchFailedError: 'Could not fetch per-tenant signing key health.',
    pageTitle: 'Signing key health (all tenants)',
    pageDescription:
      "Check the status of each tenant's signing key provider (Local / Database / VaultTransit) and its active kid across the fleet.",
    reloadAriaLabel: 'Reload the list',
    tableHeaderTenant: 'Tenant',
    tableHeaderProvider: 'Provider',
    tableHeaderActiveKid: 'Active kid',
    tableHeaderKeyCount: 'JWKS key count',
    tableHeaderProviderStatus: 'Provider status',
    healthy: 'Healthy',
    unreachable: 'Unreachable',
    noTenantsNotice: 'There are no tenants.',
  },
)
