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
  return takeNotice(noticeKey(domain))
}

const listNoticeKey = () => `idmagic:entra-list-notice:${tenantURL(ENTRA_LIST_PATH)}`

// 解除が確定した後だけ通知を預け、一覧へページ全体で移る。解除したドメインの詳細はもうないためである。
export function returnToEntraList(message: string) {
  try {
    sessionStorage.setItem(listNoticeKey(), message)
  } catch {
    // 通知を預けられなくても、解除は済んでいるので一覧へ移る。
  }
  window.location.assign(tenantURL(ENTRA_LIST_PATH))
}

export function takeEntraListNotice(): string {
  return takeNotice(listNoticeKey())
}

function takeNotice(key: string): string {
  try {
    const message = sessionStorage.getItem(key) ?? ''
    sessionStorage.removeItem(key)
    return message
  } catch {
    return ''
  }
}
