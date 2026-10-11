import { describe, expect, it } from 'bun:test'
import { verifyAgentGuidance, type GuidanceDocument } from './agent-guidance.ts'

// 文章と説明順は契約ではない。参照、コマンドと記録フィールドだけを共通にする。
const guidance: GuidanceDocument[] = [
  {
    file: 'AGENTS.md',
    source:
      'コードの変更時に docs/development/coding-style.md を読む。\ndocs/development/task-context.md に従い mise run brief -- item --target tools を使う。',
  },
  {
    file: '.agents/skills/spec-change/SKILL.md',
    source:
      'spec/modules/<module>/{models,main}.tsp\ndocs/modules/<module>/<group>/<feature>/README.md\nacceptance.feature.md\ndocs/development/task-context.md\nmise run brief -- item --concern contract',
  },
  {
    file: '.agents/skills/update-design/SKILL.md',
    source:
      'docs/README.md\ndocs/design/application/backend.md\ndocs/modules/<module>/README.md\ndocs/development/task-context.md\nmise run brief -- item --concern boundary',
  },
  {
    file: '.agents/skills/implement-work-item/SKILL.md',
    source:
      '編集に先立ち docs/development/coding-style.md を読む。\nリファクタリングの前に成功を確認する。\n受け入れ境界と単体境界で失敗を観測し、結果を記録する。\nrisk-based-v4 と fault_model を宣言する。\ndocs/development/task-context.md\nmise run brief -- item --target tools',
  },
]

const declaredTasks = new Set(['brief', 'check-work-items', 'verify'])
const implementation = '.agents/skills/implement-work-item/SKILL.md'
const verify = (documents: GuidanceDocument[]) => verifyAgentGuidance(documents, declaredTasks)
const change = (file: string, rewrite: (source: string) => string) =>
  guidance.map((document) => ({
    ...document,
    source: document.file === file ? rewrite(document.source) : document.source,
  }))

describe('verifyAgentGuidance', () => {
  it('文書参照と実行契約を保つ言い換えと説明順の変更を受理する', () => {
    expect(verify(guidance)).toEqual([])
    expect(
      verify(
        guidance.map((document) => ({
          ...document,
          source: document.source.replace('mise run brief', 'mise run -c brief'),
        })),
      ),
    ).toEqual([])
    // 説明は別の順番にも英語にもできる。特定の RED、GREEN、refactor の順序を求めない。
    expect(verify(change(implementation, (source) => `refactor\nGREEN\nRED\n${source}`))).toEqual(
      [],
    )
    expect(
      verify(change(implementation, (source) => source.replace('編集に先立ち', 'Before editing'))),
    ).toEqual([])
  })

  it('各入口から文脈選択の一次情報源への参照が消えたら拒否する', () => {
    for (const document of guidance) {
      expect(
        verify(
          change(document.file, (source) => source.replace('docs/development/task-context.md', '')),
        ),
      ).toContainEqual({
        file: document.file,
        message: 'is missing required marker: docs/development/task-context.md',
      })
    }
  })

  it('コードを変更する入口からコーディング規約への参照が消えたら拒否する', () => {
    for (const file of ['AGENTS.md', implementation]) {
      expect(
        verify(change(file, (source) => source.replace('docs/development/coding-style.md', ''))),
      ).toContainEqual({
        file,
        message: 'is missing required marker: docs/development/coding-style.md',
      })
    }
  })

  it('必須の文書と実行オプションと構造化フィールドの欠落を拒否する', () => {
    for (const [file, markers] of [
      ['AGENTS.md', ['--target']],
      [
        '.agents/skills/spec-change/SKILL.md',
        [
          'spec/modules/<module>/{models,main}.tsp',
          'docs/modules/<module>/<group>/<feature>/README.md',
          'acceptance.feature.md',
          '--concern contract',
        ],
      ],
      [
        '.agents/skills/update-design/SKILL.md',
        [
          'docs/README.md',
          'docs/design/application/backend.md',
          'docs/modules/<module>/README.md',
          '--concern boundary',
        ],
      ],
      [implementation, ['risk-based-v4', 'fault_model', '--target']],
    ] as const) {
      for (const marker of markers) {
        expect(verify(change(file, (source) => source.replace(marker, '')))).toContainEqual({
          file,
          message: `is missing required marker: ${marker}`,
        })
      }
    }
  })

  it('必須の手順ファイル全体が欠落していても拒否する', () => {
    expect(verify(guidance.filter((document) => document.file !== implementation))).toContainEqual({
      file: implementation,
      message: 'is missing required marker: risk-based-v4',
    })
  })

  it('各入口から brief の呼び出しが消えたら拒否する', () => {
    for (const document of guidance) {
      expect(
        verify(
          change(document.file, (source) => source.replace('mise run brief', 'mise run verify')),
        ),
      ).toContainEqual({ file: document.file, message: 'is missing required mise task: brief' })
    }
  })

  it('廃止した仕様形式と証拠契約への参照を拒否する', () => {
    expect(
      verify(change(implementation, (source) => `${source}\nSPECIFICATION.md\nrisk-based-v1`)),
    ).toEqual([
      { file: implementation, message: 'references the retired SPECIFICATION.md format' },
      { file: implementation, message: 'references a retired evidence policy' },
    ])
  })

  it('集約コマンドと改行継続も照合し、未定義タスクを拒否する', () => {
    expect(
      verify(
        change(implementation, (source) => `${source}\nmise run -c check-work-items ::: verify`),
      ),
    ).toEqual([])
    expect(
      verify(
        change(
          implementation,
          (source) => `${source}\nmise run -c check-work-items ::: \\\n  missing-task`,
        ),
      ),
    ).toContainEqual({
      file: implementation,
      message: 'calls an undeclared mise task: missing-task',
    })
  })
})
