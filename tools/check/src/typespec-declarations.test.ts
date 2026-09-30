import { describe, expect, it } from 'bun:test'
import { typeSpecDeclarations } from './typespec-declarations.ts'

const texts = (source: string): Record<string, string> =>
  Object.fromEntries(typeSpecDeclarations(source).map(({ name, text }) => [name, text]))

describe('typeSpecDeclarations', () => {
  it('名前空間の内側の宣言を、種類を問わず列挙する', () => {
    const source = [
      'namespace Demo {',
      'namespace Operations {',
      'model Task { id: string; }',
      'union TaskError { NotFound, Conflict }',
      'enum TaskState { ready, done }',
      'scalar TaskId extends string;',
      'alias Tasks = Task[];',
      'op StartTask(@body body: Task): Task | TaskError;',
      '}',
      '}',
    ].join('\n')
    expect(typeSpecDeclarations(source).map(({ name }) => name)).toEqual([
      'Task',
      'TaskError',
      'TaskState',
      'TaskId',
      'Tasks',
      'StartTask',
    ])
  })

  it('空白と改行だけが違う宣言を、同じ本文として読む', () => {
    const compact = 'model Task { id: string; name?: string; }'
    const spread = ['model Task {', '  id:   string;', '', '  name?: string;', '}'].join('\n')
    expect(texts(spread).Task).toBe(texts(compact).Task)
  })

  it('項目の追加と型の変更を、本文の違いとして残す', () => {
    const base = texts('model Task { id: string; }').Task
    expect(texts('model Task { id: string; name: string; }').Task).not.toBe(base)
    expect(texts('model Task { id: int32; }').Task).not.toBe(base)
  })

  it('直前のデコレーターを本文に含め、前の宣言のものは含めない', () => {
    const source = [
      'model Task { id: string; }',
      '@route("/tasks")',
      '@post',
      'op StartTask(): Task;',
    ].join('\n')
    const found = texts(source)
    expect(found.StartTask).toContain('@ route')
    expect(found.StartTask).toContain('"/tasks"')
    expect(found.Task).not.toContain('route')
  })

  it('コメントを本文から除く', () => {
    const commented = ['// 説明', 'model Task {', '  id: string; /* 識別子 */', '}'].join('\n')
    expect(texts(commented).Task).toBe(texts('model Task { id: string; }').Task)
  })

  it('文字列の中の波括弧、セミコロン、宣言の語で宣言を区切らない', () => {
    const source = [
      '@doc("The model } was renamed; op Ghost(): void; see { there")',
      'model Task { id: string; }',
      '@doc("""',
      'model Phantom {}',
      '""")',
      'op StartTask(): { @statusCode _: 200 } | Task;',
    ].join('\n')
    const found = texts(source)
    expect(Object.keys(found)).toEqual(['Task', 'StartTask'])
    expect(found.StartTask).toContain('| Task')
    expect(found.StartTask).toContain('model Phantom')
  })

  it('文字列の中の空白は正規化しない', () => {
    expect(texts('@doc("a  b")\nmodel Task {}').Task).not.toBe(
      texts('@doc("a b")\nmodel Task {}').Task,
    )
  })

  it('ブロックで終わらないモデルをセミコロンまで読む', () => {
    const found = texts('model Page is List<{ id: string }>;\nmodel Task {}')
    expect(Object.keys(found)).toEqual(['Page', 'Task'])
    expect(found.Page).toContain('>')
  })
})
