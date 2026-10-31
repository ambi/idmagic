import { useRouterState } from '@tanstack/react-router'

// 成功の通知は 1 回の遷移の結果なので、共有や再読み込みで再現する URL ではなく履歴の状態で運ぶ。
type AuthorizationNotice = 'published' | 'tuple-added'

declare module '@tanstack/history' {
  interface HistoryState {
    authorizationNotice?: AuthorizationNotice
  }
}

export function useAuthorizationNotice(): AuthorizationNotice | undefined {
  return useRouterState({ select: (state) => state.location.state.authorizationNotice })
}
