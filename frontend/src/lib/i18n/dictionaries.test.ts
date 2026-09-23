import { describe, expect, it } from 'bun:test'
import { resolve } from 'node:path'
import { untranslatedKeys } from './dictionary'
import { SUPPORTED_LOCALES } from './locale'

const sourceRoot = resolve(import.meta.dirname, '../..')

type Dictionary = Parameters<typeof untranslatedKeys>[0]

function isDictionary(value: unknown): value is Dictionary {
  return (
    typeof value === 'object' &&
    value !== null &&
    SUPPORTED_LOCALES.every((locale) => locale in value)
  )
}

describe('every *.i18n.ts dictionary', () => {
  // defineDictionary は ja の欠けた訳を en で埋めて画面に出すので、漏れは型エラーにも画面の
  // 空白にもならない。ここで全辞書の欠落を集め、漏れを CI で止める。
  it('translates every ja key without falling back to en', async () => {
    const files = await Array.fromAsync(new Bun.Glob('**/*.i18n.ts').scan({ cwd: sourceRoot }))
    const untranslated: string[] = []
    const filesWithoutDictionary: string[] = []

    for (const file of files) {
      const exports: Record<string, unknown> = await import(resolve(sourceRoot, file))
      const dictionaries = Object.entries(exports).filter((entry): entry is [string, Dictionary] =>
        isDictionary(entry[1]),
      )
      if (dictionaries.length === 0) filesWithoutDictionary.push(file)
      for (const [name, dictionary] of dictionaries) {
        for (const key of untranslatedKeys(dictionary)) untranslated.push(`${file}#${name}.${key}`)
      }
    }

    expect(files.length).toBeGreaterThan(0)
    expect(filesWithoutDictionary).toEqual([])
    expect(untranslated).toEqual([])
  })
})
