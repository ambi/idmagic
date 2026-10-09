import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import { afterAll, describe, expect, it } from 'bun:test'
import { parseLogicalArchitecture } from '../../check/src/boundary-fitness.ts'
import {
  changeCoupling,
  pathClassifier,
  renderCouplingReport,
  type CommitChange,
} from './coupling.ts'
import { readFirstParentChanges } from './git-history.ts'

const architecture = parseLogicalArchitecture(
  [
    '| モジュール | 公開方式 | 公開パッケージ | Go パッケージ | 責務 |',
    '| --- | --- | --- | --- | --- |',
    '| [System](s.md) | 組み立て地点 | なし | `backend/cmd` | Wiring |',
    ...['a', 'b', 'c', 'd', 'e', 'f'].map(
      (name) =>
        `| [${name.toUpperCase()}](${name}.md) | \`legacy\` | \`domain\` または \`ports\` の区画 | \`backend/${name}\` | ${name} |`,
    ),
    '',
  ].join('\n'),
)

const classify = pathClassifier({
  architecture,
  queryDirectories: new Set(['backend/a/db_postgres']),
  generatedDirectories: new Set(['backend/a/db_postgres']),
})

function commit(id: string, ...paths: string[]): CommitChange {
  return { id, paths }
}

describe('pathClassifier', () => {
  it('counts production Go and sqlc query inputs of a module only', () => {
    expect(
      [
        'backend/a/usecases/x.go',
        'backend/a/db_postgres/q.sql',
        'backend/a/db_postgres/q.sql.go',
        'backend/a/db_postgres/models.go',
        'backend/a/usecases/x_test.go',
        'backend/shared/clock/x.go',
        'backend/cmd/idmagic/main.go',
        'docs/design/x.md',
        'backend/gone/x.go',
        'backend/b/old/q.sql',
      ].map((path) => classify(path).kind),
    ).toEqual([
      'module',
      'module',
      'ignored',
      'ignored',
      'ignored',
      'ignored',
      'ignored',
      'ignored',
      'unclassified',
      'unclassified',
    ])
  })
})

describe('changeCoupling', () => {
  it('counts single changes, pairs, and both denominators', () => {
    const report = changeCoupling(
      [
        commit('1', 'backend/a/x.go'),
        commit('2', 'backend/a/x.go', 'backend/b/x.go'),
        commit('3', 'backend/a/y.go', 'backend/b/y.go', 'docs/x.md'),
        commit('4', 'backend/b/z.go'),
        commit('5', 'docs/only.md'),
      ],
      classify,
      6,
    )

    expect(report).toEqual({
      scanned: 5,
      counted: 4,
      withoutModule: 1,
      unclassifiedPaths: 0,
      excludeAt: 6,
      excluded: [],
      pairs: [{ first: 'A', second: 'B', firstChanges: 3, secondChanges: 3, together: 2 }],
    })
  })

  it('excludes a wide commit from both the pair counts and the module counts', () => {
    const report = changeCoupling(
      [
        commit('wide', ...['a', 'b', 'c', 'd', 'e', 'f'].map((name) => `backend/${name}/x.go`)),
        commit('pair', 'backend/a/x.go', 'backend/b/x.go'),
      ],
      classify,
      6,
    )

    expect(report.excluded).toEqual([{ id: 'wide', modules: 6 }])
    expect(report.pairs).toEqual([
      { first: 'A', second: 'B', firstChanges: 1, secondChanges: 1, together: 1 },
    ])
  })

  it('counts a rename under both paths once per module and reports unclassified paths', () => {
    const report = changeCoupling(
      [commit('rename', 'backend/a/x.go', 'backend/b/x.go', 'backend/a/y.go', 'backend/gone/z.go')],
      classify,
      6,
    )

    expect(report.pairs).toEqual([
      { first: 'A', second: 'B', firstChanges: 1, secondChanges: 1, together: 1 },
    ])
    expect(report.unclassifiedPaths).toBe(1)
  })

  it('renders the scan conditions, the exclusions, a short history, and both ratios', () => {
    const report = changeCoupling(
      [
        commit('c1', 'backend/a/x.go', 'backend/b/x.go'),
        commit('c2', 'backend/a/x.go'),
        commit('wide', ...['a', 'b', 'c'].map((name) => `backend/${name}/x.go`)),
      ],
      classify,
      3,
    )

    expect(renderCouplingReport(report, { revision: 'abc', limit: 500 })).toBe(
      [
        'revision: abc',
        'scan: first-parent, up to 500 commit(s), exclude commits changing 3 or more modules',
        'commits: 3 scanned, 2 counted, 0 without a module change, 1 excluded',
        'unclassified paths: 0',
        'history: only 3 commit(s) exist below the revision',
        'excluded: wide (3 modules)',
        '',
        'A\tB\tn(A)\tn(B)\tn(A,B)\tn(A,B)/n(A)\tn(A,B)/n(B)',
        'A\tB\t2\t1\t1\t0.50\t1.00',
      ].join('\n'),
    )
  })
})

const cleanup: string[] = []
afterAll(async () => {
  for (const path of cleanup) await rm(path, { recursive: true, force: true })
})

describe('readFirstParentChanges', () => {
  it('reads the root commit, a merge as one first-parent diff, and both paths of a rename', async () => {
    const root = await mkdtemp(join(tmpdir(), 'change-coupling-'))
    cleanup.push(root)
    const git = (...args: string[]) => {
      const result = Bun.spawnSync(
        ['git', '-c', 'user.name=t', '-c', 'user.email=t@example.com', ...args],
        { cwd: root },
      )
      if (result.exitCode !== 0) throw new Error(result.stderr.toString())
      return result.stdout.toString().trim()
    }
    const put = async (path: string, text: string) => {
      await mkdir(dirname(join(root, path)), { recursive: true })
      await writeFile(join(root, path), text)
    }
    git('init', '-q', '-b', 'main')
    await put('backend/a/x.go', 'package a\n// long enough content for rename detection\n')
    git('add', '-A')
    git('commit', '-q', '-m', 'root')
    git('checkout', '-q', '-b', 'topic')
    await put('backend/b/y.go', 'package b\n')
    git('add', '-A')
    git('commit', '-q', '-m', 'topic one')
    await put('backend/c/z.go', 'package c\n')
    git('add', '-A')
    git('commit', '-q', '-m', 'topic two')
    git('checkout', '-q', 'main')
    git('merge', '-q', '--no-ff', '-m', 'merge', 'topic')
    git('mv', 'backend/a/x.go', 'backend/b/x.go')
    git('commit', '-q', '-m', 'rename')

    const history = readFirstParentChanges(root, 'HEAD', 10)

    expect(history.revision).toBe(git('rev-parse', 'HEAD'))
    expect(history.commits.map((commit) => [...commit.paths].sort())).toEqual([
      ['backend/a/x.go', 'backend/b/x.go'],
      ['backend/b/y.go', 'backend/c/z.go'],
      ['backend/a/x.go'],
    ])
  })
})
