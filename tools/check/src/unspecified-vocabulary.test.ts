import { describe, expect, it } from 'bun:test'
import {
  collectErrorCodes,
  collectEventTypes,
  compareWithDebt,
  findUnspecified,
  specifiedTexts,
} from './unspecified-vocabulary.ts'

const TYPESPEC = [
  '@doc("The User is scheduled for deletion. RFC 9457 Problem Details with `type` = `urn:idmagic:error:user_pending_deletion`.")',
  'model UserPendingDeletionError is ProblemDetails;',
  '',
  '@doc("A user attribute is invalid. RFC 9457 Problem Details with `type` = `urn:idmagic:error:invalid_attribute`.")',
  'model InvalidUserAttributeError is ProblemDetails;',
  '',
  '@doc("A group attribute is invalid. RFC 9457 Problem Details with `type` = `urn:idmagic:error:invalid_attribute`.")',
  'model InvalidGroupAttributeError is ProblemDetails;',
  '',
  '@doc("Rejected with `urn:idmagic:error:quota_exceeded` when the tenant is full.")',
  'op createUser(): void;',
].join('\n')

const EVENTS = [
  'func (e *UserDisabled) EventType() string    { return "UserDisabled" }',
  'func (e *UserDisabled) OccurredAt() time.Time { return e.At }',
  'func (e *GroupCreated) EventType() string { return "GroupCreated" }',
].join('\n')

const SPECIFICATION = [
  '# ユーザー',
  '',
  '## 状態遷移',
  '',
  '### UserLifecycle',
  '',
  '| From | Event | Guard | To | Effects |',
  '|---|---|---|---|---|',
  '| Active | UserDisabled | — | Disabled |  |',
  '',
  '## 操作',
  '',
  '### REQ-DEMO-001 削除を予約した User の無効化を拒否する',
  '',
  '- User が `PendingDeletion` の間は、無効化を 409 と `user_pending_deletion` で拒否する。',
  '',
  '### 属性',
  '',
  '属性の誤りは InvalidUserAttributeError で拒否するが、この節は要件ではない。GroupCreated も同じ。',
].join('\n')

describe('collectErrorCodes', () => {
  it('reads every error code with the models whose documentation names it', () => {
    expect(collectErrorCodes(TYPESPEC)).toEqual(
      new Map([
        ['user_pending_deletion', ['UserPendingDeletionError']],
        ['invalid_attribute', ['InvalidUserAttributeError', 'InvalidGroupAttributeError']],
        ['quota_exceeded', []],
      ]),
    )
  })
})

describe('collectEventTypes', () => {
  it('reads the names that EventType returns', () => {
    expect(collectEventTypes(EVENTS)).toEqual(['UserDisabled', 'GroupCreated'])
  })
})

describe('specifiedTexts', () => {
  it('separates requirement bodies from the transition tables and ignores other prose', () => {
    const texts = specifiedTexts(SPECIFICATION)
    expect(texts.requirements.join('\n')).toContain('user_pending_deletion')
    expect(texts.requirements.join('\n')).not.toContain('InvalidUserAttributeError')
    expect(texts.transitions.join('\n')).toContain('UserDisabled')
    expect(texts.transitions.join('\n')).not.toContain('GroupCreated')
  })
})

describe('findUnspecified', () => {
  const texts = specifiedTexts(SPECIFICATION)

  it('names error codes and events that no requirement mentions', () => {
    expect(
      findUnspecified(
        { errors: collectErrorCodes(TYPESPEC), events: collectEventTypes(EVENTS) },
        texts,
      ),
    ).toEqual([
      { kind: 'error', name: 'invalid_attribute' },
      { kind: 'error', name: 'quota_exceeded' },
      { kind: 'event', name: 'GroupCreated' },
    ])
  })

  it('accepts an error mentioned by its model name', () => {
    const errors = new Map([['invalid_attribute', ['InvalidUserAttributeError']]])
    const mentioned = specifiedTexts(
      '### REQ-DEMO-002 属性を検証する\n\n- 違反は InvalidUserAttributeError で拒否する。\n',
    )
    expect(findUnspecified({ errors, events: [] }, mentioned)).toEqual([])
  })

  it('does not count an error code that only a transition table names', () => {
    const errors = new Map([['user_pending_deletion', []]])
    const onlyTable = specifiedTexts(
      '| From | Event | Guard | To | Effects |\n|---|---|---|---|---|\n| A | user_pending_deletion | — | B |  |\n',
    )
    expect(findUnspecified({ errors, events: [] }, onlyTable)).toEqual([
      { kind: 'error', name: 'user_pending_deletion' },
    ])
  })

  it('matches whole words only', () => {
    const mentioned = specifiedTexts('### REQ-DEMO-003 無効化\n\n- UserDisabledAt を記録する。\n')
    expect(findUnspecified({ errors: new Map(), events: ['UserDisabled'] }, mentioned)).toEqual([
      { kind: 'event', name: 'UserDisabled' },
    ])
  })
})

describe('compareWithDebt', () => {
  it('separates new violations from debt entries that are no longer violations', () => {
    expect(
      compareWithDebt(
        [
          { kind: 'error', name: 'invalid_attribute' },
          { kind: 'event', name: 'GroupCreated' },
        ],
        { errors: ['invalid_attribute', 'user_not_found'], events: [] },
      ),
    ).toEqual({
      fresh: [{ kind: 'event', name: 'GroupCreated' }],
      stale: [{ kind: 'error', name: 'user_not_found' }],
    })
  })
})
