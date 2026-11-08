import { detailURL } from './AdminWorkloadIdentityShared'

const noticeKey = (bundleID: string) => `idmagic:workload-binding-notice:${detailURL(bundleID)}`

export function returnToBundle(bundleID: string, message: string) {
  sessionStorage.setItem(noticeKey(bundleID), message)
  window.location.assign(detailURL(bundleID))
}

export function takeBundleNotice(bundleID: string): string {
  const key = noticeKey(bundleID)
  const message = sessionStorage.getItem(key) ?? ''
  sessionStorage.removeItem(key)
  return message
}
