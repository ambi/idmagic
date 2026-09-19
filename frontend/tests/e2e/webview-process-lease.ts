type Closable = {
  close(): void
}

export function createWebViewProcessLease(
  resetProcess: () => void,
  settleProcess: () => Promise<void> = async () => {},
) {
  let activeViews = 0
  let processReady = Promise.resolve()

  return {
    async waitUntilReady(): Promise<void> {
      await processReady
    },

    hold<T extends Closable>(view: T): T {
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
              resetProcess()
              processReady = settleProcess()
              return
            }
            value.call(target)
          }
        },
      })
    },
  }
}
