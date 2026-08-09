import {
  IconArrowRight,
  IconCheckupList,
  IconKey,
  IconShieldCheck,
  IconUsers,
} from '@tabler/icons-react'
import { tenantURL } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import type { TenantQuota, TenantUsage } from '../../types'
import { adminDashboardDictionary } from './AdminDashboardPage.i18n'

export function AdminDashboardPage({
  actorUsername,
  userCount,
  activeUserCount,
  disabledUserCount,
  clientCount,
  grantedConsentCount,
  quota,
  usage,
}: {
  actorUsername?: string
  userCount: number
  activeUserCount: number
  disabledUserCount: number
  clientCount: number
  grantedConsentCount: number
  quota?: TenantQuota
  usage?: TenantUsage
}) {
  const t = useDictionary(adminDashboardDictionary)
  const activeRate = userCount > 0 ? Math.round((activeUserCount / userCount) * 100) : 0

  // テナントのセキュリティ状態を評価する擬似スコア
  // ユーザー有効率や、クライアント登録、同意付与などを加味して算出
  const securityScore = Math.min(
    100,
    Math.max(
      40,
      Math.round(
        activeRate * 0.8 + (grantedConsentCount > 0 ? 10 : 0) + (clientCount > 0 ? 10 : 0),
      ),
    ),
  )

  return (
    <AdminShell
      active="dashboard"
      actorUsername={actorUsername}
      title={t.title}
      description={t.description}
    >
      <section
        className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4"
        aria-label={t.summarySectionLabel}
      >
        <DashboardMetricCard
          label={t.totalUsersLabel}
          value={userCount}
          icon={IconUsers}
          tone="blue"
          extra={
            <p className="mt-1.5 text-xs text-slate-500">
              {t.activeRateLabel.replace('{rate}', String(activeRate))} ·{' '}
              {t.disabledLabel.replace('{count}', String(disabledUserCount))}
            </p>
          }
        />
        <DashboardMetricCard
          label={t.registeredApplicationsLabel}
          value={clientCount}
          icon={IconKey}
          tone="violet"
          extra={<p className="mt-1.5 text-xs text-slate-500">{t.clientDescription}</p>}
        />
        <DashboardMetricCard
          label={t.grantedConsentsLabel}
          value={grantedConsentCount}
          icon={IconCheckupList}
          tone="green"
          extra={<p className="mt-1.5 text-xs text-slate-500">{t.consentDescription}</p>}
        />
        <DashboardMetricCard
          label="Security score"
          value={securityScore}
          icon={IconShieldCheck}
          tone="amber"
          extra={<p className="mt-1.5 text-xs text-slate-500">{t.tenantStatusValue}</p>}
        />
      </section>

      <Card className="mt-8 p-6">
        <h2 className="text-sm font-semibold text-slate-900">{t.recommendedSecurityHeading}</h2>
        <p className="mt-0.5 text-xs text-slate-500">{t.recommendedSecurityDescription}</p>
        <ul className="mt-3 divide-y divide-slate-100 border-t border-slate-100">
          <SecurityTaskCard
            title={t.mfaTaskTitle}
            description={t.mfaTaskDescription}
            href={tenantURL('/admin/sign-in-policy')}
            actionLabel={t.setPolicyAction}
          />
          <SecurityTaskCard
            title={t.federationTaskTitle}
            description={t.federationTaskDescription}
            href={tenantURL('/admin/federation/entra')}
            actionLabel={t.configureFederationAction}
          />
        </ul>
      </Card>

      {usage && (
        <Card className="mt-8 p-6">
          <h2 className="text-sm font-semibold text-slate-900">{t.quotaUsageHeading}</h2>
          <p className="mt-0.5 text-xs text-slate-500">{t.quotaUsageDescription}</p>
          <dl className="mt-3 grid grid-cols-2 gap-x-8 gap-y-4 border-t border-slate-100 pt-4 text-sm sm:grid-cols-4">
            <QuotaItem label="Users" value={usage.users} limit={quota?.users} />
            <QuotaItem label="Groups" value={usage.groups} limit={quota?.groups} />
            <QuotaItem label="Apps" value={usage.applications} limit={quota?.applications} />
            <QuotaItem label="Clients" value={usage.oauth2_clients} limit={quota?.oauth2_clients} />
          </dl>
        </Card>
      )}
    </AdminShell>
  )
}

function QuotaItem({ label, value, limit }: { label: string; value: number; limit?: number }) {
  return (
    <div>
      <dt className="text-xs text-slate-500">{label}</dt>
      <dd className="font-semibold text-slate-900">
        {value} <span className="font-normal text-slate-400">{limit ? `/ ${limit}` : ''}</span>
      </dd>
    </div>
  )
}

export function DashboardMetricCard({
  label,
  value,
  icon: Icon,
  tone,
  extra,
}: {
  label: string
  value: number
  icon: typeof IconUsers
  tone: 'blue' | 'green' | 'violet' | 'amber'
  extra?: React.ReactNode
}) {
  const tones = {
    blue: 'text-accent-foreground',
    green: 'text-emerald-700',
    violet: 'text-slate-500',
    amber: 'text-amber-700',
  }
  return (
    <Card className="p-5">
      <div className="flex items-center gap-2">
        <Icon size={16} stroke={1.8} className={tones[tone]} aria-hidden="true" />
        <p className="text-xs font-semibold text-slate-500">{label}</p>
      </div>
      <p className="mt-1.5 text-3xl font-semibold tracking-tight text-slate-950">{value}</p>
      {extra}
    </Card>
  )
}

export function SecurityTaskCard({
  title,
  description,
  href,
  actionLabel,
}: {
  title: string
  description: string
  href: string
  actionLabel: string
}) {
  return (
    <li className="flex flex-col gap-1 py-4 sm:flex-row sm:items-center sm:justify-between sm:gap-6">
      <div>
        <h3 className="text-sm font-medium text-slate-900">{title}</h3>
        <p className="mt-0.5 text-xs text-slate-500">{description}</p>
      </div>
      <a
        href={href}
        className="inline-flex shrink-0 items-center gap-1 text-xs font-semibold text-accent-foreground hover:underline"
      >
        {actionLabel}
        <IconArrowRight size={12} aria-hidden="true" />
      </a>
    </li>
  )
}
