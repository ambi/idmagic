type Closable = {
  close(): void
}

// Chrome バックエンドは 1 つのプロセスで Cookie と保存領域を共有する。開いている view が
// なくなった時点で reset に最後の view を渡し、次のテストへ状態を持ち越させない。
// reset はその view を閉じる責任も持つ。次の view は reset が終わるまで待つ。
export function createWebViewProcessLease<T extends Closable>(
  reset: (lastView: T) => Promise<void>,
) {
  let activeViews = 0
  let processReady = Promise.resolve()

  return {
    async waitUntilReady(): Promise<void> {
      await processReady
    },

    hold(view: T): T {
      activeViews += 1
      let released = false

      return new Proxy(view, {
        get(target, property) {
          const value = Reflect.get(target, property, target)
          if (property !== 'close' || typeof value !== 'function') return value

          return () => {
            if (released) return
            released = true
            activeViews -= 1
            if (activeViews === 0) {
              processReady = reset(target)
              return
            }
            value.call(target)
          }
        },
      })
    },
  }
}
