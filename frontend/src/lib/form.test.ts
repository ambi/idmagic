import { describe, expect, it } from 'bun:test'
import {
  cidrList,
  codePointLength,
  containsPemBlock,
  type FieldSchema,
  formSchema,
  integerInRange,
  isCIDR,
  isRFC3339,
  optionalList,
  optionalNonNegativeInteger,
  optionalPositiveInteger,
  optionalText,
  optionalUrl,
  pemText,
  positiveInteger,
  requiredJson,
  requiredList,
  requiredSecret,
  requiredText,
  requiredUrl,
  scopeText,
  urlList,
  when,
} from './form'

function parse<O>(schema: FieldSchema<O>, value: string) {
  const result = schema.safeParse(value)
  if (result.success) return { data: result.data }
  const issue = result.error.issues[0]
  return { problem: issue?.code === 'custom' ? issue.params?.problem : issue }
}

describe('requiredText', () => {
  it('trims the value it sends', () => {
    expect(parse(requiredText(10), '  taro  ')).toEqual({ data: 'taro' })
  })

  it('rejects an empty value and a value of only spaces', () => {
    expect(parse(requiredText(10), '')).toEqual({ problem: { kind: 'required' } })
    expect(parse(requiredText(10), ' \t ')).toEqual({ problem: { kind: 'required' } })
  })

  // REQ-SYSTEM-028 の数え方。絵文字は UTF-16 で 2 単位だが 1 コードポイントである。
  it('accepts the limit in code points and rejects one more', () => {
    expect(parse(requiredText(3), '😀😀😀')).toEqual({ data: '😀😀😀' })
    expect(parse(requiredText(3), '😀😀😀😀')).toEqual({ problem: { kind: 'tooLong', max: 3 } })
  })

  it('counts the length after trimming', () => {
    expect(parse(requiredText(3), '  abc  ')).toEqual({ data: 'abc' })
  })
})

describe('optionalText', () => {
  it('sends nothing for an empty value', () => {
    expect(parse(optionalText(10), '   ')).toEqual({ data: undefined })
  })

  it('accepts the limit in code points and rejects one more', () => {
    expect(parse(optionalText(2), ' 日本 ')).toEqual({ data: '日本' })
    expect(parse(optionalText(2), '日本語')).toEqual({ problem: { kind: 'tooLong', max: 2 } })
  })
})

describe('optionalPositiveInteger', () => {
  it('sends nothing for an empty value', () => {
    expect(parse(optionalPositiveInteger(), '   ')).toEqual({ data: undefined })
  })

  it('accepts 1 and larger integers after trimming', () => {
    expect(parse(optionalPositiveInteger(), '1')).toEqual({ data: 1 })
    expect(parse(optionalPositiveInteger(), ' 1800 ')).toEqual({ data: 1800 })
  })

  it.each(['0', '-10', '1.5', '1.0', '05', '+5', '1e3', 'abc'])('rejects %s', (value) => {
    expect(parse(optionalPositiveInteger(), value)).toEqual({
      problem: { kind: 'positiveInteger' },
    })
  })
})

describe('positiveInteger', () => {
  it('requires a value', () => {
    expect(parse(positiveInteger(), ' ')).toEqual({ problem: { kind: 'required' } })
  })

  it('accepts 1 and rejects 0', () => {
    expect(parse(positiveInteger(), '1')).toEqual({ data: 1 })
    expect(parse(positiveInteger(), '0')).toEqual({ problem: { kind: 'positiveInteger' } })
  })
})

describe('integerInRange', () => {
  it('accepts both ends and rejects one past each', () => {
    const depth = integerInRange(0, 16)
    expect(parse(depth, '0')).toEqual({ data: 0 })
    expect(parse(depth, '16')).toEqual({ data: 16 })
    expect(parse(depth, '17')).toEqual({ problem: { kind: 'integerRange', min: 0, max: 16 } })
    expect(parse(integerInRange(1, 10), '0')).toEqual({
      problem: { kind: 'integerRange', min: 1, max: 10 },
    })
  })

  it.each(['-1', '1.5', '01', '1e1', 'x'])('rejects %s', (value) => {
    expect(parse(integerInRange(0, 100), value).problem).toEqual({
      kind: 'integerRange',
      min: 0,
      max: 100,
    })
  })

  it('requires a value only when asked', () => {
    expect(parse(integerInRange(1, 10), ' ')).toEqual({ data: undefined })
    expect(parse(integerInRange(1, 10, { required: true }), ' ')).toEqual({
      problem: { kind: 'required' },
    })
  })
})

describe('optionalNonNegativeInteger', () => {
  it('accepts 0 and sends nothing for an empty value', () => {
    expect(parse(optionalNonNegativeInteger(), '0')).toEqual({ data: 0 })
    expect(parse(optionalNonNegativeInteger(), ' 42 ')).toEqual({ data: 42 })
    expect(parse(optionalNonNegativeInteger(), '')).toEqual({ data: undefined })
  })

  it.each(['-1', '1.5', '01', 'x'])('rejects %s', (value) => {
    expect(parse(optionalNonNegativeInteger(), value)).toEqual({
      problem: { kind: 'nonNegativeInteger' },
    })
  })
})

describe('isRFC3339', () => {
  it.each([
    '2026-12-31T23:59:59Z',
    '2026-12-31T23:59:59.123456789Z',
    '2026-12-31T23:59:59+09:00',
    '2024-02-29T00:00:00-05:30',
  ])('accepts %s', (value) => {
    expect(isRFC3339(value)).toBe(true)
  })

  it.each([
    '2026-12-31',
    '2026-12-31 23:59:59Z',
    '2026-12-31t23:59:59z',
    '2026-02-30T00:00:00Z',
    '2025-02-29T00:00:00Z',
    '2026-12-31T24:00:00Z',
    '2026-12-31T23:59:59',
    '2026-12-31T23:59:59+0900',
  ])('rejects %s', (value) => {
    expect(isRFC3339(value)).toBe(false)
  })
})

describe('optionalList with maxItems', () => {
  it('accepts the maximum number of items and rejects one more', () => {
    const split = (value: string) => value.split(',').filter(Boolean)
    expect(parse(optionalList(split, 2), 'a,b')).toEqual({ data: ['a', 'b'] })
    expect(parse(optionalList(split, 2), 'a,b,c')).toEqual({
      problem: { kind: 'tooMany', max: 2 },
    })
  })
})

// 期待値は RFC 6749 §3.3 の scope-token（%x21 / %x23-5B / %x5D-7E）から決めた。
describe('scopeText', () => {
  it('joins tokens with one space and sends nothing for an empty value', () => {
    expect(parse(scopeText(), '  openid\tprofile \n email ')).toEqual({
      data: 'openid profile email',
    })
    expect(parse(scopeText(), '   ')).toEqual({ data: undefined })
    expect(parse(scopeText(), '!#[]~ catalog:read')).toEqual({ data: '!#[]~ catalog:read' })
  })

  it.each(['say"hi', 'back\\slash', 'スコープ'])(
    'names %s, which is not a scope-token',
    (entry) => {
      expect(parse(scopeText(), `openid ${entry}`)).toEqual({
        problem: { kind: 'invalidScope', entry },
      })
    },
  )
})

describe('requiredSecret', () => {
  it('keeps surrounding spaces and rejects only an empty value', () => {
    expect(parse(requiredSecret(), ' pass phrase ')).toEqual({ data: ' pass phrase ' })
    expect(parse(requiredSecret(), '')).toEqual({ problem: { kind: 'required' } })
  })
})

describe('requiredList', () => {
  const split = (value: string) => value.split(/\s+/).filter(Boolean)

  it('requires at least one entry', () => {
    expect(parse(requiredList(split), ' \n ')).toEqual({ problem: { kind: 'required' } })
    expect(parse(requiredList(split), 'a b')).toEqual({ data: ['a', 'b'] })
  })
})

describe('when', () => {
  it('skips an inactive field and sends nothing for it', () => {
    expect(parse(when(false, requiredText()), '')).toEqual({ data: undefined })
    expect(parse(when(true, requiredText()), '')).toEqual({ problem: { kind: 'required' } })
  })
})

// 既定の規則の期待値は zog の URL()（url.Parse が通り、Scheme と Host が空でない）から、
// https の規則は tokens_jose.ValidateJWKSURI から決めた。
describe('URL rules', () => {
  it.each([
    'https://app.example.com/callback',
    'http://localhost:3000/cb',
    'com.example.app://callback',
    'https://[2001:db8::1]:8443/cb',
    'https://user@app.example.com/cb',
    'https://app.example.com/cb#frag',
    'https://app.example.com:99999/cb',
    'https://例え.jp/cb',
  ])('accepts %s as an absolute URL', (value) => {
    expect(parse(requiredUrl(), value)).toEqual({ data: value })
  })

  it.each([
    ['app.example.com/callback', 'スキームがない'],
    ['https:app.example.com', 'authority がない'],
    ['https:///callback', 'ホストが空'],
    ['https://app example.com/cb', 'ホストの空白'],
    ['https://app.example.com:80a/cb', '数字でないポート'],
    ['https://app.example.com/\u0007', '制御文字'],
    ['https://[2001:db8::1/cb', '閉じない角括弧'],
    ['1https://app.example.com', 'スキームが数字で始まる'],
  ])('rejects %s (%s)', (value) => {
    expect(parse(requiredUrl(), value)).toEqual({ problem: { kind: 'invalidUrl' } })
  })

  it('requires https, a host, no userinfo, and no fragment for the https rule', () => {
    const https = requiredUrl({ https: true })
    expect(parse(https, 'https://app.example.com/jwks.json')).toEqual({
      data: 'https://app.example.com/jwks.json',
    })
    expect(parse(https, 'HTTPS://app.example.com/jwks.json').problem).toBeUndefined()
    // Go の url.Parse は空のフラグメントを Fragment == "" と読む。
    expect(parse(https, 'https://app.example.com/jwks.json#').problem).toBeUndefined()
    for (const value of [
      'http://app.example.com/jwks.json',
      'https://user@app.example.com/jwks.json',
      'https://app.example.com/jwks.json#keys',
      'https://:443/jwks.json',
    ]) {
      expect(parse(https, value)).toEqual({ problem: { kind: 'httpsUrl' } })
    }
  })

  it('sends nothing for an empty optional URL', () => {
    expect(parse(optionalUrl(), '  ')).toEqual({ data: undefined })
    expect(parse(optionalUrl(), 'nope')).toEqual({ problem: { kind: 'invalidUrl' } })
  })

  it('names the first entry of a list that is not a URL', () => {
    const split = (value: string) => value.split(/\s+/).filter(Boolean)
    expect(parse(urlList(split), 'https://a.example/cb nope https://b.example')).toEqual({
      problem: { kind: 'invalidUrl', entry: 'nope' },
    })
    expect(parse(urlList(split, { required: true }), ' ')).toEqual({
      problem: { kind: 'required' },
    })
    expect(parse(urlList(split), ' ')).toEqual({ data: [] })
  })

  // 往復：組み立てた絶対 URL を受理し、スキームの区切りを壊した値を拒む。
  it('accepts generated absolute URLs and rejects them without the scheme separator', () => {
    let seed = 7
    const random = (bound: number) => {
      seed = (seed * 1103515245 + 12345) % 2 ** 31
      return seed % bound
    }
    const pick = (chars: string, length: number) =>
      Array.from({ length }, () => chars[random(chars.length)]).join('')
    for (let n = 0; n < 300; n++) {
      const scheme = pick('abcdefghijklmnopqrstuvwxyz', 1) + pick('abcz0189+.-', random(6))
      const host = `${pick('abcdefghij0123456789-', 1 + random(10))}.example`
      const port = random(2) === 0 ? '' : `:${random(70000)}`
      const path = `/${pick('abc/._~!$&()*+,;=:@%20', random(12))}`
      const query = random(2) === 0 ? '' : `?${pick('a=b&c', random(8))}`
      const url = `${scheme}://${host}${port}${path}${query}`
      expect(parse(requiredUrl(), url).problem).toBeUndefined()
      expect(parse(requiredUrl(), url.replace('://', ':/')).problem).toEqual({ kind: 'invalidUrl' })
    }
  })
})

describe('containsPemBlock', () => {
  const body =
    'MIIBszCCAVmgAwIBAgIUQ0FCQ0FCQ0FCQ0FCQ0FCQ0FCQ0Ew\nCgYIKoZIzj0EAwIwDzENMAsGA1UEAwwEdGVzdDAe'

  it('accepts a block, surrounding text, and PEM headers', () => {
    expect(
      containsPemBlock(`-----BEGIN CERTIFICATE-----\n${body}\n-----END CERTIFICATE-----`),
    ).toBe(true)
    expect(
      containsPemBlock(
        `subject=CN=test\n-----BEGIN CERTIFICATE-----\r\n${body}\r\n-----END CERTIFICATE-----\n`,
      ),
    ).toBe(true)
    expect(
      containsPemBlock(
        `-----BEGIN CERTIFICATE-----\nProc-Type: 4,ENCRYPTED\n\n${body}\n-----END CERTIFICATE-----`,
      ),
    ).toBe(true)
  })

  it.each([
    ['', '空'],
    [body, '囲みがない'],
    [
      `-----BEGIN CERTIFICATE-----\n${body}\n-----END PUBLIC KEY-----`,
      '見出しと終わりの種類が違う',
    ],
    ['-----BEGIN CERTIFICATE-----\n\n-----END CERTIFICATE-----', '本文が空'],
    ['-----BEGIN CERTIFICATE-----\nnot base64!\n-----END CERTIFICATE-----', 'base64 でない本文'],
  ])('rejects %#: %s', (value) => {
    expect(containsPemBlock(value)).toBe(false)
  })

  it('checks the certificate only when it is required', () => {
    expect(parse(pemText(false), 'anything')).toEqual({ data: 'anything' })
    expect(parse(pemText(true), ' ')).toEqual({ problem: { kind: 'required' } })
    expect(parse(pemText(true), 'anything')).toEqual({ problem: { kind: 'invalidPem' } })
  })
})

describe('requiredJson', () => {
  it('parses JSON and rejects a value that does not parse', () => {
    expect(parse(requiredJson(), ' {"type":"object"} ')).toEqual({ data: { type: 'object' } })
    expect(parse(requiredJson(), '{type: object}')).toEqual({ problem: { kind: 'invalidJson' } })
    expect(parse(requiredJson(), '')).toEqual({ problem: { kind: 'required' } })
  })
})

describe('codePointLength', () => {
  it('counts a surrogate pair as one', () => {
    expect(codePointLength('a😀')).toBe(2)
    expect('a😀'.length).toBe(3)
  })
})

describe('cidrList', () => {
  it('splits lines and drops blank lines and surrounding spaces', () => {
    expect(parse(cidrList(), ' 10.0.0.0/8 \n\n2001:db8::/32\n')).toEqual({
      data: ['10.0.0.0/8', '2001:db8::/32'],
    })
  })

  it('names the first entry that is not a CIDR', () => {
    expect(parse(cidrList(), '10.0.0.0/8\n10.0.0.1\n300.0.0.0/8')).toEqual({
      problem: { kind: 'invalidCidr', entry: '10.0.0.1' },
    })
  })
})

// 期待値は Go の netip.ParsePrefix の受理と拒否から決めた。サーバーはこの関数で検証する。
describe('isCIDR', () => {
  it.each([
    '0.0.0.0/0',
    '10.0.0.0/8',
    '192.0.2.1/32',
    '192.0.2.1/24',
    '::/0',
    '::1/128',
    '2001:db8::/32',
    '2001:DB8:0:0:0:0:0:1/64',
    'fe80::/10',
    '::ffff:192.0.2.1/128',
    '1:2:3:4:5:6:192.0.2.1/96',
    '1:2:3:4:5:6:7::/112',
    '::2:3:4:5:6:7:8/0',
  ])('accepts %s', (value) => {
    expect(isCIDR(value)).toBe(true)
  })

  it.each([
    ['', '空'],
    ['10.0.0.0', '接頭辞長がない'],
    ['10.0.0.0/', '接頭辞長が空'],
    ['10.0.0.0/33', 'IPv4 の接頭辞長が 32 を超える'],
    ['10.0.0.0/08', '接頭辞長の先頭の 0'],
    ['10.0.0.0/+8', '接頭辞長の符号'],
    ['10.0.0/8', 'オクテットが 3 つ'],
    ['10.0.0.0.0/8', 'オクテットが 5 つ'],
    ['10.0.0.256/8', 'オクテットが 255 を超える'],
    ['10.0.0.01/8', 'オクテットの先頭の 0'],
    ['10.0.0.0x1/8', '16 進のオクテット'],
    [' 10.0.0.0/8', '前の空白'],
    ['::/129', 'IPv6 の接頭辞長が 128 を超える'],
    ['2001:db8:::/32', ':: の後の余分なコロン'],
    ['2001::db8::/32', ':: が 2 つ'],
    [':2001:db8::/32', '先頭の単独のコロン'],
    ['2001:db8:/32', '末尾の単独のコロン'],
    ['1:2:3:4:5:6:7:8::/64', ':: が 0 組を表す'],
    ['1:2:3:4:5:6:7:8:9/64', '組が 9 つ'],
    ['1:2:3:4:5:6:7/64', '組が 7 つで省略がない'],
    ['12345::/16', '組が 5 桁'],
    ['g::/16', '16 進でない文字'],
    ['fe80::1%eth0/64', 'ゾーン'],
    ['1:2:3:4:5:192.0.2.1/96', '省略のない IPv4 表記が最後の 2 組の位置にない'],
    ['::ffff:192.0.2.01/128', '埋め込んだ IPv4 のオクテットの先頭の 0'],
    ['::192.0.2.1:1/128', 'IPv4 表記の後の組'],
  ])('rejects %s (%s)', (value) => {
    expect(isCIDR(value)).toBe(false)
  })

  // 往復：正しく組み立てた値を受理し、接頭辞長を範囲の外へ変えた値を拒否する。
  it('accepts generated prefixes and rejects them with a prefix length out of range', () => {
    let seed = 1
    const random = (bound: number) => {
      seed = (seed * 1103515245 + 12345) % 2 ** 31
      return seed % bound
    }
    for (let n = 0; n < 500; n++) {
      const octets = Array.from({ length: 4 }, () => String(random(256)))
      const v4 = octets.join('.')
      expect(isCIDR(`${v4}/${random(33)}`)).toBe(true)
      expect(isCIDR(`${v4}/${33 + random(967)}`)).toBe(false)

      const groups = Array.from({ length: 8 }, () =>
        random(3) === 0 ? '0' : random(0x10000).toString(16),
      )
      const start = random(8)
      const length = 1 + random(8 - start)
      const compressed =
        random(2) === 0
          ? groups.join(':')
          : `${groups.slice(0, start).join(':')}::${groups.slice(start + length).join(':')}`
      const v6 = random(2) === 0 ? compressed : compressed.toUpperCase()
      expect(isCIDR(`${v6}/${random(129)}`)).toBe(true)
      expect(isCIDR(`${v6}/${129 + random(871)}`)).toBe(false)
    }
  })
})

describe('jitless', () => {
  // CSP が unsafe-eval を許さないので、生成を試す呼び出しそのものが違反として報告される。
  it('parses an object without constructing a function', () => {
    const original = globalThis.Function
    const constructed: unknown[][] = []
    globalThis.Function = new Proxy(original, {
      construct(target, args) {
        constructed.push(args)
        return Reflect.construct(target, args)
      },
      apply(target, thisArg, args) {
        constructed.push(args)
        return Reflect.apply(target, thisArg, args)
      },
    })
    try {
      const schema = formSchema({ name: requiredText(10), note: optionalText(10) })
      expect(schema.safeParse({ name: 'taro', note: '' }).success).toBe(true)
    } finally {
      globalThis.Function = original
    }
    expect(constructed).toEqual([])
  })
})
