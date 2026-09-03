import { describe, expect, it } from 'bun:test'
import { isControlPlaneAccount, type AccountContextResponse } from './-guards'

const account = (
  realm: string | undefined,
  roles: string[] | undefined,
): AccountContextResponse => ({
  csrf_token: 'csrf',
  sub: 'operator',
  realm,
  roles,
})

describe('isControlPlaneAccount', () => {
  it('制御面 realm の system_admin だけを許可する', () => {
    expect(isControlPlaneAccount(account('default', ['system_admin']))).toBeTrue()
  })

  it('制御面以外の realm に保存された system_admin を拒否する', () => {
    expect(isControlPlaneAccount(account('acme', ['system_admin']))).toBeFalse()
  })

  it('制御面 realm でも system_admin がなければ拒否する', () => {
    expect(isControlPlaneAccount(account('default', ['admin']))).toBeFalse()
    expect(isControlPlaneAccount(account('default', undefined))).toBeFalse()
  })
})
