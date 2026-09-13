import { describe, expect, it } from 'bun:test'
import { verifyMutationCanary } from './mutation-canary.ts'

const mutation = (type: string, status: string) => ({ type, status })

describe('verifyMutationCanary', () => {
  it('accepts the expected survivor and compile failure', () => {
    expect(
      verifyMutationCanary({
        files: [
          {
            mutations: [
              mutation('CONDITIONALS_BOUNDARY', 'LIVED'),
              mutation('ARITHMETIC_BASE', 'NOT VIABLE'),
            ],
          },
        ],
      }),
    ).toEqual([])
  })

  it('rejects a runner that counts the compile failure as killed', () => {
    expect(
      verifyMutationCanary({
        files: [
          {
            mutations: [
              mutation('CONDITIONALS_BOUNDARY', 'LIVED'),
              mutation('ARITHMETIC_BASE', 'KILLED'),
            ],
          },
        ],
      }),
    ).toContain('expected an ARITHMETIC_BASE mutant with status NOT VIABLE')
  })

  it('rejects a runner that loses the known survivor', () => {
    expect(
      verifyMutationCanary({
        files: [
          {
            mutations: [
              mutation('CONDITIONALS_BOUNDARY', 'KILLED'),
              mutation('ARITHMETIC_BASE', 'NOT VIABLE'),
            ],
          },
        ],
      }),
    ).toContain('expected a CONDITIONALS_BOUNDARY mutant with status LIVED')
  })

  it('rejects a malformed report', () => {
    expect(verifyMutationCanary({})).toEqual(['mutation report does not contain a files array'])
  })
})
