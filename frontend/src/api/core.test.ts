import { describe, it, expect, beforeEach, afterEach, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../test/globals'
import {
  tenantBasePath,
  tenantLocalPath,
  tenantRouterPath,
  tenantURL,
  validReturnTo,
  base64URL,
  AuthenticationAPIError,
  UnauthenticatedError,
  setBearerTokenProvider,
  adminRequest,
  request,
  requestPage,
} from './core'

describe('core api utils', () => {
  const originalLocation = window.location

  beforeEach(() => {
    stubGlobal('location', {
      ...originalLocation,
      pathname: '/realms/test-tenant/dashboard',
      origin: 'http://localhost:5173',
    })
  })

  afterEach(() => {
    restoreGlobals()
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
      stubGlobal('location', {
        ...originalLocation,
        pathname: '/realms/test-tenant',
      })
      expect(tenantLocalPath()).toBe('/')
    })
  })

  describe('tenantRouterPath', () => {
    it('should remove the tenant prefix from a router destination', () => {
      expect(tenantRouterPath('/realms/test-tenant/admin/users')).toBe('/admin/users')
      expect(tenantRouterPath('/realms/test-tenant')).toBe('/')
    })

    it('should preserve a destination without a tenant prefix', () => {
      expect(tenantRouterPath('/admin/users')).toBe('/admin/users')
    })
  })

  describe('tenantURL', () => {
    it('should prepend tenant base path to input path', () => {
      expect(tenantURL('/admin/users')).toBe('/realms/test-tenant/admin/users')
    })

    it('uses the default path tenant from the local development root', () => {
      stubGlobal('location', {
        ...originalLocation,
        hostname: 'localhost',
        port: '5173',
        pathname: '/',
        origin: 'http://localhost:5173',
      })
      expect(tenantURL('/authorize')).toBe('/realms/default/authorize')
    })

    it('uses the default path tenant from the E2E development root', () => {
      stubGlobal('location', {
        ...originalLocation,
        hostname: 'localhost',
        port: '5174',
        pathname: '/',
        origin: 'http://localhost:5174',
      })
      expect(tenantURL('/authorize')).toBe('/realms/default/authorize')
    })

    it('uses the default path tenant from the docker-compose frontend root', () => {
      stubGlobal('location', {
        ...originalLocation,
        hostname: 'localhost',
        port: '8080',
        pathname: '/',
        origin: 'http://localhost:8080',
      })
      expect(tenantURL('/authorize')).toBe('/realms/default/authorize')
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
      const mockFetch = mock().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockData),
      })
      stubGlobal('fetch', mockFetch)
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
      const mockFetch = mock().mockResolvedValue({
        ok: false,
        status: 401,
        json: () => Promise.resolve({ error: 'unauthorized', message: 'Not logged in' }),
      })
      stubGlobal('fetch', mockFetch)
      setBearerTokenProvider(() => null)

      await expect(request('/secure-api')).rejects.toThrow(UnauthenticatedError)
    })

    it('should throw AuthenticationAPIError on other non-ok status', async () => {
      const mockFetch = mock().mockResolvedValue({
        ok: false,
        status: 500,
        json: () => Promise.resolve({ error: 'server_error', message: 'Internal error' }),
      })
      stubGlobal('fetch', mockFetch)

      await expect(request('/error-api')).rejects.toThrow(AuthenticationAPIError)
    })

    it('uses RFC 9457 detail and type suffix for API errors', async () => {
      stubGlobal(
        'fetch',
        mock().mockResolvedValue({
          ok: false,
          status: 500,
          json: () =>
            Promise.resolve({
              type: 'urn:idmagic:error:internal_server_error',
              title: 'Internal server error',
              status: 500,
              detail: 'The import result could not be loaded.',
            }),
        }),
      )

      await expect(request('/problem-api')).rejects.toMatchObject({
        name: 'AuthenticationAPIError',
        message: 'The import result could not be loaded.',
        code: 'internal_server_error',
      })
    })

    it('uses RFC 9457 Problem Details for unauthenticated errors', async () => {
      stubGlobal(
        'fetch',
        mock().mockResolvedValue({
          ok: false,
          status: 401,
          json: () =>
            Promise.resolve({
              type: 'urn:idmagic:error:unauthorized',
              title: 'Unauthorized',
              status: 401,
              detail: 'The session has expired.',
            }),
        }),
      )

      await expect(request('/problem-auth-api')).rejects.toMatchObject({
        name: 'UnauthenticatedError',
        message: 'The session has expired.',
        code: 'unauthorized',
      })
    })
  })

  describe('requestPage', () => {
    const paginationHeaders = (headers?: HeadersInit) =>
      new Headers({
        'Pagination-Total-Items': '0',
        'Pagination-Total-Pages': '0',
        'Pagination-Current-Page': '0',
        'Pagination-Page-Size': '50',
        ...headers,
      })

    it('extracts the cursor from a Link rel="next" header', async () => {
      const mockData = { users: [{ id: '1' }] }
      const mockFetch = mock().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockData),
        headers: paginationHeaders({
          Link: '<http://localhost/realms/test-tenant/api/admin/v1/users?limit=50&cursor=abc123>; rel="next"',
        }),
      })
      stubGlobal('fetch', mockFetch)

      const page = await requestPage('/api/admin/v1/users')
      expect(page.body).toEqual(mockData)
      expect(page.nextCursor).toBe('abc123')
    })

    it('extracts previous and next cursors from a bidirectional Link header', async () => {
      const mockFetch = mock().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ users: [] }),
        headers: paginationHeaders({
          Link: '<http://localhost/api/users?cursor=before>; rel="prev", <http://localhost/api/users?cursor=after>; rel="next"',
        }),
      })
      stubGlobal('fetch', mockFetch)

      const page = await requestPage('/api/admin/v1/users')
      expect(page.previousCursor).toBe('before')
      expect(page.nextCursor).toBe('after')
    })

    it('extracts first/last targets and exact pagination metadata', async () => {
      const mockFetch = mock().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ users: [] }),
        headers: paginationHeaders({
          Link: '<http://localhost/api/users?limit=50>; title="start"; rel="first next", </api/users?limit=50&cursor=end>; rel=last',
          'Pagination-Total-Items': '105',
          'Pagination-Total-Pages': '3',
          'Pagination-Current-Page': '2',
          'Pagination-Page-Size': '50',
        }),
      })
      stubGlobal('fetch', mockFetch)

      const page = await requestPage('/api/admin/v1/users')
      expect(page.hasFirst).toBeTrue()
      expect(page.lastCursor).toBe('end')
      expect(page.totalItems).toBe(105)
      expect(page.totalPages).toBe(3)
      expect(page.currentPage).toBe(2)
      expect(page.pageSize).toBe(50)
    })

    it('returns null nextCursor when there is no Link header', async () => {
      const mockFetch = mock().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ users: [] }),
        headers: paginationHeaders(),
      })
      stubGlobal('fetch', mockFetch)

      const page = await requestPage('/api/admin/v1/users')
      expect(page.nextCursor).toBeNull()
    })

    it('returns null nextCursor when the Link header has no rel="next"', async () => {
      const mockFetch = mock().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ users: [] }),
        headers: paginationHeaders({ Link: '<http://localhost/whatever>; rel="prev"' }),
      })
      stubGlobal('fetch', mockFetch)

      const page = await requestPage('/api/admin/v1/users')
      expect(page.nextCursor).toBeNull()
    })

    it('rejects missing or malformed required pagination metadata', async () => {
      const mockFetch = mock()
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ users: [] }),
          headers: new Headers(),
        })
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ users: [] }),
          headers: paginationHeaders({ 'Pagination-Total-Items': '-1' }),
        })
      stubGlobal('fetch', mockFetch)

      await expect(requestPage('/api/admin/v1/users')).rejects.toThrow(
        'Missing or invalid Pagination-Total-Items',
      )
      await expect(requestPage('/api/admin/v1/users')).rejects.toThrow(
        'Missing or invalid Pagination-Total-Items',
      )
    })
  })
})
