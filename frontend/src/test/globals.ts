// vitest の `vi.stubGlobal` / `vi.unstubAllGlobals` に相当する bun ネイティブヘルパー。
// bun:test にはグローバル差し替え API が無いため、globalThis のプロパティ記述子を
// スタックで退避・復元することで同じ意味論（後入れ先出しの復元、元が未定義なら削除）を再現する。
interface StubEntry {
  key: PropertyKey
  descriptor: PropertyDescriptor | undefined
}

const stubs: StubEntry[] = []

export function stubGlobal(key: PropertyKey, value: unknown): void {
  const target = globalThis as Record<PropertyKey, unknown>
  stubs.push({ key, descriptor: Object.getOwnPropertyDescriptor(target, key) })
  Object.defineProperty(target, key, {
    value,
    writable: true,
    configurable: true,
    enumerable: true,
  })
}

export function restoreGlobals(): void {
  const target = globalThis as Record<PropertyKey, unknown>
  while (stubs.length > 0) {
    const entry = stubs.pop()
    if (!entry) continue
    const { key, descriptor } = entry
    if (descriptor) {
      Object.defineProperty(target, key, descriptor)
    } else {
      Reflect.deleteProperty(target, key)
    }
  }
}
