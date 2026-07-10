import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import {
  tenantBasePath,
  tenantLocalPath,
  tenantURL,
  validReturnTo,
  base64URL,
  AuthenticationAPIError,
  UnauthenticatedError,
  setBearerTokenProvider,
  adminRequest,
  request,
} from './core'

describe('core api utils', () => {
  const originalLocation = window.location

  beforeEach(() => {
    vi.stubGlobal('location', {
      ...originalLocation,
      pathname: '/realms/test-tenant/dashboard',
      origin: 'http://localhost:5173',
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  describe('Errors', () => {
    it('should create AuthenticationAPIError with code', () => {
      const err = new AuthenticationAPIError('msg', 'ERR_CODE')
      expect(err.message).toBe('msg')
      expect(err.code).toBe('ERR_CODE')
      expect(err.name).toBe('AuthenticationAPIError')
    })

    it('should create UnauthenticatedError with code', () => {
      const err = new UnauthenticatedError('msg', 'UNAUTH_CODE')
      expect(err.message).toBe('msg')
      expect(err.code).toBe('UNAUTH_CODE')
      expect(err.name).toBe('UnauthenticatedError')
    })
  })

  describe('tenantBasePath', () => {
    it('should return the correct base path for valid tenant URL paths', () => {
      expect(tenantBasePath('/realms/my-tenant/admin')).toBe('/realms/my-tenant')
      expect(tenantBasePath('/realms/another-123-tenant')).toBe('/realms/another-123-tenant')
    })

    it('should return empty string for non-tenant paths', () => {
      expect(tenantBasePath('/admin')).toBe('')
      expect(tenantBasePath('/')).toBe('')
    })

    it('should use window.location.pathname by default', () => {
      expect(tenantBasePath()).toBe('/realms/test-tenant')
    })
  })

  describe('tenantLocalPath', () => {
    it('should return local path without tenant prefix', () => {
      expect(tenantLocalPath()).toBe('/dashboard')
    })

    it('should return slash if local path is empty', () => {
      vi.stubGlobal('location', {
        ...originalLocation,
        pathname: '/realms/test-tenant',
      })
      expect(tenantLocalPath()).toBe('/')
    })
  })

  describe('tenantURL', () => {
    it('should prepend tenant base path to input path', () => {
      expect(tenantURL('/admin/users')).toBe('/realms/test-tenant/admin/users')
    })
  })

  describe('validReturnTo', () => {
    it('should accept valid admin and wsfed paths under the tenant base', () => {
      expect(validReturnTo('/realms/test-tenant/admin')).toBe(true)
      expect(validReturnTo('/realms/test-tenant/admin/users')).toBe(true)
      expect(validReturnTo('/realms/test-tenant/wsfed')).toBe(true)
    })

    it('should reject invalid paths or external URLs', () => {
      expect(validReturnTo('http://malicious.com')).toBe(false)
      expect(validReturnTo('/realms/test-tenant/other')).toBe(false)
      expect(validReturnTo('/realms/test-tenant/admin\\escaped')).toBe(false)
      expect(validReturnTo('//malicious.com/admin')).toBe(false)
    })
  })

  describe('base64URL', () => {
    it('should encode Uint8Array to base64url correctly', () => {
      const data = new Uint8Array([0, 1, 2, 3, 4, 255])
      expect(base64URL(data)).toBe('AAECAwT_')
    })
  })

  describe('adminRequest', () => {
    it('should build request options with CSRF token', () => {
      const options = adminRequest('token123', 'POST', { foo: 'bar' })
      expect(options.method).toBe('POST')
      expect(options.headers).toEqual({
        'Content-Type': 'application/json',
        'X-CSRF-Token': 'token123',
      })
      expect(options.body).toBe(JSON.stringify({ foo: 'bar' }))
    })

    it('should omit body if undefined', () => {
      const options = adminRequest('token123', 'GET')
      expect(options.body).toBeUndefined()
    })
  })

  describe('request', () => {
    it('should fetch data successfully', async () => {
      const mockData = { success: true }
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockData),
      })
      vi.stubGlobal('fetch', mockFetch)
      setBearerTokenProvider(() => 'my-token')

      const res = await request('/test-api')
      expect(res).toEqual(mockData)
      expect(mockFetch).toHaveBeenCalledWith(
        '/realms/test-tenant/test-api',
        expect.objectContaining({
          headers: expect.objectContaining({
            Authorization: 'Bearer my-token',
          }),
        }),
      )
    })

    it('should throw UnauthenticatedError on 401', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        json: () => Promise.resolve({ error: 'unauthorized', message: 'Not logged in' }),
      })
      vi.stubGlobal('fetch', mockFetch)
      setBearerTokenProvider(() => null)

      await expect(request('/secure-api')).rejects.toThrow(UnauthenticatedError)
    })

    it('should throw AuthenticationAPIError on other non-ok status', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        json: () => Promise.resolve({ error: 'server_error', message: 'Internal error' }),
      })
      vi.stubGlobal('fetch', mockFetch)

      await expect(request('/error-api')).rejects.toThrow(AuthenticationAPIError)
    })
  })
})
