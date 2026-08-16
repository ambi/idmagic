import { listAdminAuditEvents } from '../../api'
import type { AdminAuditEvent } from '../../types'

const RECENT_REJECTION_LIMIT = 50

export type RejectionWindow = {
  events: AdminAuditEvent[]
  unavailable: boolean
  // truncated は取得窓を使い切ったこと、すなわち「これより古い拒否がまだある」ことを表す。
  truncated: boolean
}

// loadRejectionEvents は WorkloadAttestationRejected を type 完全一致で引く。専用の読み取り経路は
// 作らず、既存の監査検索を再利用する (すべての DomainEvent は監査へ写されており、この
// イベントは tenantId を持つのでテナントで絞り込まれる)。
//
// 監査の検索軸は閉じた許可リストで決まり、そこに trustBundleId は無い。したがって信頼バンドル
// 単位の絞り込みは取得後に画面側で行うしかなく、他バンドルの拒否が窓を埋めると、拒否があるのに
// 「該当なし」に見えうる。truncated を返して呼び出し側がその可能性を明示できるようにする
// (軸そのものの追加は [[wi-377-agent-and-delegation-chain-audit-axes]] の領分)。
//
// 読み込みに失敗しても信頼設定そのものの管理は続けられるべきなので、画面全体を落とさず
// 「拒否の履歴だけが読めない」ことを呼び出し側へ返す。
export async function loadRejectionEvents(): Promise<RejectionWindow> {
  try {
    const page = await listAdminAuditEvents({
      type: 'WorkloadAttestationRejected',
      limit: RECENT_REJECTION_LIMIT,
    })
    const events = page.events ?? []
    return { events, unavailable: false, truncated: events.length >= RECENT_REJECTION_LIMIT }
  } catch {
    return { events: [], unavailable: true, truncated: false }
  }
}
