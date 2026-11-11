import { tenantURL } from '../../api'

export const ENTRA_LIST_PATH = '/admin/federation/entra'
export const ENTRA_ADD_PATH = '/admin/federation/entra/new'

export function entraDomainURL(domain: string): string {
  return tenantURL(`${ENTRA_LIST_PATH}/${encodeURIComponent(domain)}`)
}

export function entraDomainEditURL(domain: string): string {
  return tenantURL(`${ENTRA_LIST_PATH}/${encodeURIComponent(domain)}/edit`)
}

const noticeKey = (domain: string) => `idmagic:entra-domain-notice:${entraDomainURL(domain)}`

// 保存が確定した後だけ通知を預け、ドメインの詳細へページ全体で移る。詳細が一度だけ読む。
export function returnToEntraDomain(domain: string, message: string) {
  try {
    sessionStorage.setItem(noticeKey(domain), message)
  } catch {
    // 通知を預けられなくても、保存は済んでいるので詳細へ移る。
  }
  window.location.assign(entraDomainURL(domain))
}

export function takeEntraDomainNotice(domain: string): string {
  try {
    const key = noticeKey(domain)
    const message = sessionStorage.getItem(key) ?? ''
    sessionStorage.removeItem(key)
    return message
  } catch {
    return ''
  }
}
