import { tenantURL } from '../../api'
import { CopyableValue } from '../../components/CopyableValue'
import { useDictionary } from '../../lib/i18n'
import type { McpResourceServer } from '../../types'
import { adminMcpResourceServersDictionary } from './AdminMcpResourceServersPage.i18n'

// 接続の準備は保存済みの登録から案内し、未保存のフォームや評価用の秘密を配布しない。
export function ManagedMcpConnectionGuide({ server }: { server: McpResourceServer }) {
  const t = useDictionary(adminMcpResourceServersDictionary)
  if (!server.hosting) return null

  return (
    <section className="mt-4 flex flex-col gap-3 text-sm" aria-label={t.connectionGuide}>
      <h3 className="font-semibold">{t.connectionGuide}</h3>
      <CopyableValue
        id="result-public-url"
        value={server.resource}
        copyLabel={t.copy.replace('{label}', t.publicURL)}
      />
      <p>{t.connectionProtocol}</p>
      <ol className="list-decimal space-y-2 pl-5">
        <li>{t.connectionPublish}</li>
        {server.hosting.authentication === 'token_vault' ? (
          <li>
            {t.connectionVault}{' '}
            <a className="text-blue-700 underline" href={tenantURL('/account/connections')}>
              {t.connectionServices}
            </a>
            <p className="mt-1 break-all font-mono text-xs">
              {t.connectorID}: {server.hosting.connector_id}
            </p>
          </li>
        ) : null}
        <li>{t.connectionAuthorize}</li>
      </ol>
      <p className="text-xs text-slate-600">{t.connectionUnsupported}</p>
      <p className="text-xs text-slate-600">{t.connectionCredentials}</p>
    </section>
  )
}
