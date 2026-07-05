import {
  IconArrowLeft,
  IconDownload,
  IconPlus,
  IconServerBolt,
  IconTrash,
  IconWorldShare,
} from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import {
  AuthenticationAPIError,
  configureEntraFederation,
  type ConfigureEntraFederationResponse,
  deleteWsFedRelyingParty,
  tenantURL,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Toast } from '../../components/ui/toast'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import type { WsFedRelyingParty } from '../../types'

// Entra domain federation は検証済みドメイン単位の操作。個別アプリの設定ではなく、テナント配下の
// ドメインをまとめて外部 IdP へ向ける設定なので、Application 編集とは別のテナント設定画面で扱う。
//
// 詳細→編集ポリシー (wi-126 §8): 最初に出るこの画面は「現在 federation 済みのドメイン一覧」
// (= 現在の設定) を見せるだけにし、追加 (configure) は専用画面 /admin/federation/entra/new に
// 分離する。一覧画面がそのまま追加フォームを兼ねている状態を無くす。
const ENDPOINT_LINKS = [
  {
    label: 'Federation metadata',
    href: tenantURL('/federationmetadata/2007-06/federationmetadata.xml'),
    value: '/federationmetadata/2007-06/federationmetadata.xml',
    icon: IconDownload,
  },
  {
    label: 'WS-Trust MEX',
    href: tenantURL('/trust/mex'),
    value: '/trust/mex',
    icon: IconServerBolt,
  },
  {
    label: 'WS-Trust usernamemixed',
    href: tenantURL('/trust/usernamemixed'),
    value: '/trust/usernamemixed',
    icon: IconServerBolt,
  },
]

const ADD_PATH = '/admin/federation/entra/new'
const LIST_PATH = '/admin/federation/entra'

export function AdminEntraFederationPage({
  csrfToken,
  actorUsername,
  relyingParties,
}: {
  csrfToken: string
  actorUsername?: string
  relyingParties: WsFedRelyingParty[]
}) {
  const [items, setItems] = useState(relyingParties.filter((rp) => rp.entra_profile))
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  async function handleDelete(rp: WsFedRelyingParty) {
    setError('')
    try {
      await deleteWsFedRelyingParty(csrfToken, rp.wtrealm)
      setItems((prev) => prev.filter((item) => item.wtrealm !== rp.wtrealm))
      setNotice(`${rp.entra_profile?.domain ?? rp.wtrealm} のフェデレーションを削除しました。`)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : '削除に失敗しました。')
    }
  }

  return (
    <AdminShell
      active="entra-federation"
      actorUsername={actorUsername}
      title="Entra ドメインフェデレーション"
      description="検証済みドメインを Microsoft Entra から本 IdP へフェデレーションします。ドメインごとに UPN / ImmutableID / persistent NameID の claim preset を持つ relying party を作成します。"
      actions={
        <Button asChild>
          <a href={tenantURL(ADD_PATH)}>
            <IconPlus size={16} aria-hidden="true" />
            ドメインフェデレーションを追加
          </a>
        </Button>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />

      <div className="grid gap-2 rounded-md border border-slate-200 bg-slate-50 p-3 sm:grid-cols-3">
        {ENDPOINT_LINKS.map(({ label, href, value, icon: Icon }) => (
          <a
            key={value}
            className="flex min-w-0 items-start gap-2 rounded-md border border-slate-200 bg-white p-2.5 text-xs text-slate-600 hover:border-blue-300 hover:text-blue-700"
            href={href}
          >
            <Icon size={16} className="mt-0.5 shrink-0 text-blue-600" aria-hidden="true" />
            <span className="min-w-0">
              <span className="block font-semibold text-slate-800">{label}</span>
              <span className="block truncate font-mono">{value}</span>
            </span>
          </a>
        ))}
      </div>

      {items.length === 0 ? (
        <Card className="p-8 text-center text-sm text-slate-500">
          まだフェデレーション済みのドメインがありません。「ドメインフェデレーションを追加」から作成してください。
        </Card>
      ) : (
        <div className="flex flex-col gap-3">
          {items.map((rp) => (
            <Card key={rp.wtrealm} className="flex items-start justify-between gap-3 p-4">
              <div className="flex min-w-0 items-start gap-2">
                <IconWorldShare
                  size={16}
                  className="mt-0.5 shrink-0 text-blue-600"
                  aria-hidden="true"
                />
                <div className="min-w-0 text-xs leading-5 text-slate-600">
                  <p className="font-semibold text-slate-900">{rp.entra_profile?.domain}</p>
                  <p className="truncate font-mono">{rp.wtrealm}</p>
                  <p>sourceAnchor: {rp.entra_profile?.source_anchor_attribute}</p>
                </div>
              </div>
              <Button type="button" variant="ghost" onClick={() => handleDelete(rp)}>
                <IconTrash size={16} aria-hidden="true" />
              </Button>
            </Card>
          ))}
        </div>
      )}
    </AdminShell>
  )
}

// AdminEntraFederationAddPage は詳細→編集ポリシー (wi-126 §8) に沿って一覧から分離した
// ドメイン federation の追加専用画面 (/admin/federation/entra/new)。保存すると M365 domain
// federation 用の PowerShell 手順を一度だけ表示し、一覧へ戻る導線を出す。
export function AdminEntraFederationAddPage({
  csrfToken,
  actorUsername,
}: {
  csrfToken: string
  actorUsername?: string
}) {
  const [domain, setDomain] = useState('')
  const [issuer, setIssuer] = useState('')
  const [sourceAnchor, setSourceAnchor] = useState('object_guid')
  const [replyURL, setReplyURL] = useState('https://login.microsoftonline.com/login.srf')
  const [result, setResult] = useState<ConfigureEntraFederationResponse | null>(null)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  async function handleConfigure(event: FormEvent) {
    event.preventDefault()
    setError('')
    setNotice('')
    try {
      const configured = await configureEntraFederation(csrfToken, {
        domain: domain.trim(),
        issuer_uri: issuer.trim() || undefined,
        source_anchor_attribute: sourceAnchor.trim(),
        reply_url: replyURL.trim() || undefined,
      })
      setResult(configured)
      setNotice(`${configured.profile.domain} のドメインフェデレーションを保存しました。`)
    } catch (cause) {
      setError(
        cause instanceof AuthenticationAPIError
          ? cause.message
          : 'Entra federation の保存に失敗しました。',
      )
    }
  }

  return (
    <AdminShell
      active="entra-federation"
      actorUsername={actorUsername}
      title="ドメインフェデレーションを追加"
      description="Microsoft 365 のドメインフェデレーション向けに、検証済みドメインごとに claim preset を作成します。"
      actions={
        <Button asChild variant="outline">
          <a href={tenantURL(LIST_PATH)}>
            <IconArrowLeft size={16} aria-hidden="true" />
            フェデレーション一覧
          </a>
        </Button>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />

      <Card className="grid gap-4 p-4">
        <form className="grid gap-3 lg:grid-cols-[1fr_1fr_1fr_1fr_auto]" onSubmit={handleConfigure}>
          <div className="grid gap-1.5">
            <Label htmlFor="entra_domain">検証済み domain</Label>
            <Input
              id="entra_domain"
              value={domain}
              placeholder="contoso.com"
              onChange={(e) => setDomain(e.target.value)}
              required
            />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="entra_source_anchor">sourceAnchor 属性</Label>
            <Input
              id="entra_source_anchor"
              value={sourceAnchor}
              onChange={(e) => setSourceAnchor(e.target.value)}
              required
            />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="entra_issuer">IssuerUri</Label>
            <Input
              id="entra_issuer"
              value={issuer}
              placeholder="空なら自動生成"
              onChange={(e) => setIssuer(e.target.value)}
            />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="entra_reply">wreply URL</Label>
            <Input
              id="entra_reply"
              value={replyURL}
              onChange={(e) => setReplyURL(e.target.value)}
            />
          </div>
          <div className="flex items-end">
            <Button type="submit">保存</Button>
          </div>
        </form>
        {result ? (
          <div className="grid gap-3 rounded-md border border-slate-200 bg-slate-50 p-3 text-xs">
            <div className="grid gap-1 font-mono text-slate-700 sm:grid-cols-2">
              {Object.entries(result.powershell).map(([key, value]) => (
                <div key={key} className="min-w-0">
                  <span className="block font-sans font-semibold text-slate-600">{key}</span>
                  <span className="block truncate">{value}</span>
                </div>
              ))}
            </div>
            <Alert>
              Hybrid Azure AD Join のデバイス登録は未提供です。必要な場合は managed/PHS
              への切替または AD FS 併存を検討してください。
            </Alert>
            <div>
              <Button asChild variant="outline">
                <a href={tenantURL(LIST_PATH)}>
                  <IconArrowLeft size={16} aria-hidden="true" />
                  フェデレーション一覧へ戻る
                </a>
              </Button>
            </div>
          </div>
        ) : null}
      </Card>
    </AdminShell>
  )
}
