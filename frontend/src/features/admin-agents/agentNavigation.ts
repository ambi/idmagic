import { tenantURL } from '../../api'

const noticeKey = (agentID: string) =>
  `idmagic:agent-notice:${tenantURL(`/admin/agents/${encodeURIComponent(agentID)}`)}`

// 成功が確定した後だけ通知を預ける。ページ全体の遷移後に、対象の詳細が一度だけ読む。
export function returnToAgent(agentID: string, message: string) {
  sessionStorage.setItem(noticeKey(agentID), message)
  window.location.assign(tenantURL(`/admin/agents/${encodeURIComponent(agentID)}`))
}

export function takeAgentNotice(agentID: string): string {
  const key = noticeKey(agentID)
  const message = sessionStorage.getItem(key) ?? ''
  sessionStorage.removeItem(key)
  return message
}
