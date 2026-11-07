import { useId, useState } from 'react'
import { z } from 'zod'

// Zod は object の解析を `new Function` で生成できるかを最初の parse で試す。このアプリの CSP は
// `unsafe-eval` を許さないので生成は拒否されるが、試した呼び出しが CSP 違反として報告される。
// 画面から Zod を直接 import せずこのモジュールを通すので、ここで設定すれば最初の parse より前に効く。
z.config({ jitless: true })

/** 欄の値が受け付けられない理由。文言への変換は FieldError が辞書で行う。 */
export type FieldProblem =
  | { kind: 'required' }
  | { kind: 'tooLong'; max: number }
  | { kind: 'tooMany'; max: number }
  | { kind: 'invalidCidr'; entry: string }
  | { kind: 'positiveInteger' }
  | { kind: 'nonNegativeInteger' }
  | { kind: 'integerRange'; min: number; max: number }
  | { kind: 'dateTime' }
  /** `entry` は一覧の欄で、受け付けない最初の項目を示す。 */
  | { kind: 'invalidUrl'; entry?: string }
  | { kind: 'httpsUrl'; entry?: string }
  | { kind: 'resourceUrl'; entry?: string }
  | { kind: 'invalidScope'; entry: string }
  | { kind: 'invalidPem' }
  | { kind: 'invalidJson' }
  | { kind: 'invalid' }
  /** 画面の辞書で作った文言。共通の種類で表せない画面固有の規則に使う。 */
  | { kind: 'message'; text: string }

/** 入力 `I` を受け取り、送信する値 `O` へ変換する欄の規則。 */
export type FieldSchema<O, I = string> = z.ZodType<O, I>

type FormShape = Record<string, FieldSchema<unknown, unknown>>

/** 欄の規則をまとめたフォーム全体の規則。 */
export type FormSchema<Shape extends FormShape = FormShape> = z.ZodObject<Shape>

export function formSchema<Shape extends FormShape>(shape: Shape): FormSchema<Shape> {
  return z.object(shape)
}

/**
 * `check` が問題を返したら、その欄を受け付けない。
 * 画面固有の規則を共通の規則に重ねるときに使う。
 */
export function rule<O, I>(
  schema: FieldSchema<O, I>,
  check: (value: O) => FieldProblem | undefined,
): FieldSchema<O, I> {
  return schema.superRefine((value, ctx) => {
    const problem = check(value)
    if (problem) ctx.addIssue({ code: 'custom', message: problem.kind, params: { problem } })
  })
}

/**
 * 文字列の長さを Unicode のコードポイントの数で数える。サーバーの数え方（REQ-SYSTEM-028）と同じであり、
 * UTF-16 の単位で数える HTML の `maxLength` とは、基本多言語面の外の文字で一致しない。
 */
export function codePointLength(value: string): number {
  let count = 0
  for (const _ of value) count++
  return count
}

function lengthProblem(value: string, max: number | undefined): FieldProblem | undefined {
  return max !== undefined && codePointLength(value) > max ? { kind: 'tooLong', max } : undefined
}

/**
 * 前後の空白を除き、空なら受け付けない。
 * 長さの上限は、契約が上限を定める欄にだけ `LENGTH` の値を渡す。
 */
export function requiredText(max?: number): FieldSchema<string> {
  return rule(
    z.string().transform((value) => value.trim()),
    (value) => (value === '' ? { kind: 'required' } : lengthProblem(value, max)),
  )
}

/**
 * 前後の空白を除き、空も受け付ける。編集画面で空にして値を消す欄に使う。
 * 新規作成で空を送らない欄には `optionalText` を使う。
 */
export function text(max?: number): FieldSchema<string> {
  return rule(
    z.string().transform((value) => value.trim()),
    (value) => lengthProblem(value, max),
  )
}

/** 前後の空白を除き、空なら送信しない値（`undefined`）にする。 */
export function optionalText(max?: number): FieldSchema<string | undefined> {
  return rule(
    z.string().transform((value) => value.trim() || undefined),
    (value) => (value === undefined ? undefined : lengthProblem(value, max)),
  )
}

/**
 * 空なら受け付けない。パスワードやシークレットは前後の空白も値の一部なので削らず、
 * 長さの上限も付けない（パスワード管理ソフトから貼り付けた長い値を拒まないため）。
 */
export function requiredSecret(): FieldSchema<string> {
  return rule(z.string(), (value) => (value === '' ? { kind: 'required' } : undefined))
}

// 1 以上の十進の整数。`+5`、`05`、`1e3`、`1.0` のように数としては読めても表記の違う値は受け付けない。
const POSITIVE_INTEGER = /^[1-9][0-9]*$/

/** 1 以上の整数を数値にする。空なら required。 */
export function positiveInteger(): FieldSchema<number> {
  return rule(
    z.string().transform((value) => value.trim()),
    (value) => {
      if (value === '') return { kind: 'required' }
      return POSITIVE_INTEGER.test(value) ? undefined : { kind: 'positiveInteger' }
    },
  ).transform(Number)
}

/** 1 以上の整数を数値にする。空なら送信しない値（`undefined`）にする。 */
export function optionalPositiveInteger(): FieldSchema<number | undefined> {
  return rule(
    z.string().transform((value) => value.trim()),
    (value) =>
      value === '' || POSITIVE_INTEGER.test(value) ? undefined : { kind: 'positiveInteger' },
  ).transform((value) => (value === '' ? undefined : Number(value)))
}

const NON_NEGATIVE_INTEGER = /^(0|[1-9][0-9]*)$/

/** 0 以上の整数を数値にする。空なら送信しない値（`undefined`）にする。上限は契約が定めない欄に使う。 */
export function optionalNonNegativeInteger(): FieldSchema<number | undefined> {
  return rule(
    z.string().transform((value) => value.trim()),
    (value) =>
      value === '' || NON_NEGATIVE_INTEGER.test(value) ? undefined : { kind: 'nonNegativeInteger' },
  ).transform((value) => (value === '' ? undefined : Number(value)))
}

/**
 * `min` 以上 `max` 以下の整数を数値にする。範囲は契約の `@minValue` と `@maxValue` を渡す。
 * `required` でなければ、空は送信しない値（`undefined`）にする。
 */
export function integerInRange(
  min: number,
  max: number,
  { required = false }: { required?: boolean } = {},
): FieldSchema<number | undefined> {
  return rule(
    z.string().transform((value) => value.trim()),
    (value) => {
      if (value === '') return required ? { kind: 'required' } : undefined
      const ok = NON_NEGATIVE_INTEGER.test(value) && Number(value) >= min && Number(value) <= max
      return ok ? undefined : { kind: 'integerRange', min, max }
    },
  ).transform((value) => (value === '' ? undefined : Number(value)))
}

// Go の time.Time が JSON から読む RFC 3339 の表記。T と Z は大文字だけを受け付ける。
const RFC3339 =
  /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(\.\d{1,9})?(Z|[+-](\d{2}):(\d{2}))$/

/** RFC 3339 の日時として実在する値かを判定する。 */
export function isRFC3339(value: string): boolean {
  const match = RFC3339.exec(value)
  if (!match) return false
  const [year, month, day, hour, minute, second] = match.slice(1, 7).map(Number) as number[]
  const offsetHour = match[9] === undefined ? 0 : Number(match[9])
  const offsetMinute = match[10] === undefined ? 0 : Number(match[10])
  const date = new Date(Date.UTC(year!, month! - 1, day!))
  return (
    date.getUTCFullYear() === year &&
    date.getUTCMonth() === month! - 1 &&
    date.getUTCDate() === day &&
    hour! <= 23 &&
    minute! <= 59 &&
    second! <= 59 &&
    offsetHour <= 23 &&
    offsetMinute <= 59
  )
}

/** RFC 3339 の日時。空なら送信しない値（`undefined`）にする。 */
export function optionalDateTime(): FieldSchema<string | undefined> {
  return rule(optionalText(), (value) =>
    value === undefined || isRFC3339(value) ? undefined : { kind: 'dateTime' },
  )
}

/**
 * `split` で分けた項目の配列にする。空の配列も受け付ける。
 * `maxItems` は契約の `@maxItems` を渡す。
 */
export function optionalList(
  split: (value: string) => string[],
  maxItems?: number,
): FieldSchema<string[]> {
  return rule(z.string().transform(split), (entries) =>
    maxItems !== undefined && entries.length > maxItems
      ? { kind: 'tooMany', max: maxItems }
      : undefined,
  )
}

// RFC 6749 §3.3 の scope-token：%x21 / %x23-5B / %x5D-7E。
const SCOPE_TOKEN = /^[\x21\x23-\x5b\x5d-\x7e]+$/

/**
 * 空白で区切ったスコープを、一つの空白で連結した値にする。空なら送信しない値（`undefined`）にする。
 * サーバーは登録時に文字種を検査しないが、scope-token でない値は認可の要求で使えないので受け付けない。
 */
export function scopeText(): FieldSchema<string | undefined> {
  return rule(
    z.string().transform((value) => value.split(/\s+/).filter(Boolean)),
    (tokens) => {
      const entry = tokens.find((token) => !SCOPE_TOKEN.test(token))
      return entry === undefined ? undefined : { kind: 'invalidScope', entry }
    },
  ).transform((tokens) => (tokens.length === 0 ? undefined : tokens.join(' ')))
}

/** `split` で分けた項目が一つもなければ受け付けない。 */
export function requiredList(split: (value: string) => string[]): FieldSchema<string[]> {
  return rule(z.string().transform(split), (entries) =>
    entries.length === 0 ? { kind: 'required' } : undefined,
  )
}

/**
 * 選んだ種類や方式で使わない欄を、検査せず送信しない値（`undefined`）にする。
 * 欄の名前と型を選択によらず一定に保ち、送信値を一つの型で扱うため、形から欄を外さない。
 */
export function when<O, I>(
  active: boolean,
  schema: FieldSchema<O, I>,
): FieldSchema<O | undefined, I> {
  if (active) return schema
  return z.unknown().transform(() => undefined) as unknown as FieldSchema<undefined, I>
}

type AbsoluteUrl = {
  scheme: string
  userinfo?: string
  hostname: string
  hostport: string
  fragment: string
}

// Go の url.Parse が受け付けるホストの文字。非 ASCII は IDN として通す。
const HOST_CHARS = /^[A-Za-z0-9\-._~!$&'()*+,;=%<>"\u0080-￿]*$/

/**
 * `scheme://authority...` の形の絶対 URL を、サーバーが使う Go の `url.Parse` に近い規則で分解する。
 * 制御文字、authority の空白、ホストに使えない文字、数字でないポートを拒む。
 * ポートの範囲は Go も検査しないので見ない。
 */
export function parseAbsoluteUrl(value: string): AbsoluteUrl | undefined {
  // biome-ignore lint/suspicious/noControlCharactersInRegex: Go の url.Parse と同じく制御文字を拒む
  if (/[\u0000-\u001f\u007f]/.test(value)) return undefined
  const match = /^([A-Za-z][A-Za-z0-9+.-]*):\/\/([^/?#]*)[^#]*(?:#([\s\S]*))?$/.exec(value)
  if (!match) return undefined
  const [, scheme = '', authority = '', fragment = ''] = match
  if (/\s/.test(authority)) return undefined
  const at = authority.lastIndexOf('@')
  const hostport = authority.slice(at + 1)
  let hostname: string
  let port: string
  if (hostport.startsWith('[')) {
    const end = hostport.indexOf(']')
    if (end < 0) return undefined
    hostname = hostport.slice(1, end)
    port = hostport.slice(end + 1)
  } else {
    const colon = hostport.lastIndexOf(':')
    hostname = colon < 0 ? hostport : hostport.slice(0, colon)
    port = colon < 0 ? '' : hostport.slice(colon)
    if (!HOST_CHARS.test(hostname)) return undefined
  }
  if (port !== '' && !/^:[0-9]*$/.test(port)) return undefined
  return {
    scheme: scheme.toLowerCase(),
    userinfo: at < 0 ? undefined : authority.slice(0, at),
    hostname,
    hostport,
    fragment,
  }
}

/**
 * URL の欄の規則。既定はスキームとホストを持つ絶対 URL で、OAuth2 のリダイレクト URI と同じである。
 * `https` は JWKS URI やログアウト URI と同じく、https、ホスト、userinfo とフラグメントがないことを求める。
 * `noFragment` は RFC 8707 のリソース指示子と同じく、絶対 URL にフラグメントがないことを求める。
 */
export type UrlRule = { https?: boolean; noFragment?: boolean }

function urlProblem(value: string, options: UrlRule, entry?: string): FieldProblem | undefined {
  const url = parseAbsoluteUrl(value)
  if (!options.https) {
    const ok =
      url !== undefined && url.hostport !== '' && (!options.noFragment || url.fragment === '')
    return ok ? undefined : { kind: options.noFragment ? 'resourceUrl' : 'invalidUrl', entry }
  }
  const ok =
    url !== undefined &&
    url.scheme === 'https' &&
    url.hostname !== '' &&
    url.userinfo === undefined &&
    url.fragment === ''
  return ok ? undefined : { kind: 'httpsUrl', entry }
}

/** 前後の空白を除いた URL。空なら required。 */
export function requiredUrl(options: UrlRule = {}, max?: number): FieldSchema<string> {
  return rule(requiredText(max), (value) => urlProblem(value, options))
}

/** 前後の空白を除いた URL。空なら送信しない値（`undefined`）にする。 */
export function optionalUrl(options: UrlRule = {}, max?: number): FieldSchema<string | undefined> {
  return rule(optionalText(max), (value) =>
    value === undefined ? undefined : urlProblem(value, options),
  )
}

/** `split` で分けた URL の一覧。受け付けない最初の項目を示す。`required` なら空の一覧を拒む。 */
export function urlList(
  split: (value: string) => string[],
  { required = false, ...options }: UrlRule & { required?: boolean } = {},
): FieldSchema<string[]> {
  return rule(z.string().transform(split), (entries) => {
    if (required && entries.length === 0) return { kind: 'required' }
    for (const entry of entries) {
      const problem = urlProblem(entry, options, entry)
      if (problem) return problem
    }
    return undefined
  })
}

/**
 * Go の `pem.Decode` が読む PEM のブロックを一つ含むかを判定する。
 * 前後の文字は `pem.Decode` が読み飛ばすので許す。DER の中身はサーバーが x509 として検査する。
 */
export function containsPemBlock(value: string): boolean {
  const match = /-----BEGIN ([^\r\n-]*)-----\r?\n([\s\S]*?)-----END \1-----/.exec(value)
  if (!match) return false
  const lines = (match[2] ?? '').split(/\r?\n/).map((line) => line.trim())
  // 先頭の `Name: value` の見出しと空行は pem.Decode が見出しとして読む。
  let i = 0
  while (i < lines.length && /^[^:\s]+:/.test(lines[i] ?? '')) i++
  const body = lines.slice(i).join('')
  return body !== '' && /^[A-Za-z0-9+/]*={0,2}$/.test(body)
}

/** JSON として読める文字列を、その値にする。空なら required。 */
export function requiredJson(): FieldSchema<unknown> {
  return rule(requiredText(), (value) => {
    try {
      JSON.parse(value)
      return undefined
    } catch {
      return { kind: 'invalidJson' }
    }
  }).transform((value) => JSON.parse(value) as unknown)
}

/**
 * PEM の証明書の欄。空なら送信しない値（`undefined`）にする。
 * `required` のときだけ、空と PEM でない値を拒む。サーバーも証明書を使う設定のときだけ解析するためである。
 */
export function pemText(required: boolean): FieldSchema<string | undefined> {
  return rule(optionalText(), (value) => {
    if (!required) return undefined
    if (value === undefined) return { kind: 'required' }
    return containsPemBlock(value) ? undefined : { kind: 'invalidPem' }
  })
}

/** 行ごとの CIDR を、前後の空白と空行を除いた配列にする。CIDR でない行があれば最初の行を示す。 */
export function cidrList(): FieldSchema<string[]> {
  return rule(
    z.string().transform((value) =>
      value
        .split('\n')
        .map((entry) => entry.trim())
        .filter((entry) => entry !== ''),
    ),
    (entries) => {
      const entry = entries.find((candidate) => !isCIDR(candidate))
      return entry === undefined ? undefined : { kind: 'invalidCidr', entry }
    },
  )
}

/**
 * サーバーが使う Go の `netip.ParsePrefix` と同じ文法で CIDR を判定する。
 * IPv4 のオクテットと接頭辞長は先頭の 0 を許さず、IPv6 は `::` の省略と末尾の IPv4 表記を許し、
 * ゾーンは許さない。ホスト部のビットはサーバーが落とすので、立っていても受け付ける。
 */
export function isCIDR(value: string): boolean {
  const slash = value.lastIndexOf('/')
  if (slash < 0) return false
  const address = value.slice(0, slash)
  const bits = value.slice(slash + 1)
  if (!/^(0|[1-9][0-9]{0,2})$/.test(bits)) return false
  if (address.includes(':')) return isIPv6(address) && Number(bits) <= 128
  return isIPv4(address) && Number(bits) <= 32
}

function isIPv4(value: string): boolean {
  const octets = value.split('.')
  return (
    octets.length === 4 &&
    octets.every((octet) => /^(0|[1-9][0-9]{0,2})$/.test(octet) && Number(octet) <= 255)
  )
}

function isHexDigit(char: string | undefined): boolean {
  return char !== undefined && /^[0-9a-fA-F]$/.test(char)
}

function isIPv6(value: string): boolean {
  // groups は解析した 16 ビットの組の数で、末尾の IPv4 表記は 2 組に数える。
  let groups = 0
  let ellipsis = false
  let i = 0
  if (value.startsWith('::')) {
    ellipsis = true
    i = 2
  }
  while (i < value.length) {
    let end = i
    while (end - i < 5 && isHexDigit(value[end])) end++
    const digits = end - i
    if (digits === 0 || digits > 4) return false
    if (value[end] === '.') {
      // 末尾の IPv4 表記は、省略がなければ最後の 2 組の位置にだけ置ける。
      if ((!ellipsis && groups !== 6) || groups + 2 > 8) return false
      if (!isIPv4(value.slice(i))) return false
      groups += 2
      break
    }
    groups++
    i = end
    if (i === value.length) break
    if (value[i] !== ':' || i + 1 === value.length) return false
    i++
    if (value[i] === ':') {
      if (ellipsis) return false
      ellipsis = true
      i++
    }
    if (groups === 8 && i < value.length) return false
  }
  // `::` は少なくとも 1 組の 0 を表す。
  return ellipsis ? groups < 8 : groups === 8
}

type Values<S extends FormSchema> = z.input<S>
type FieldName<S extends FormSchema> = keyof Values<S> & string

export type FormValidation<S extends FormSchema = FormSchema> = {
  /** 全欄が妥当なときだけ、送信する値を持つ。送信ボタンの無効化にも使う。 */
  data: z.output<S> | undefined
  /** 表示する問題。値が初期値から変わるか、フォーカスが外れるまでは返さない。 */
  problem(name: FieldName<S>): FieldProblem | undefined
  /** 入力欄に渡す属性。問題を表示する間だけ `aria-invalid` とエラー文への参照を付ける。 */
  fieldProps(name: FieldName<S>): {
    'aria-invalid'?: true
    'aria-describedby'?: string
    onBlur: () => void
  }
  errorId(name: FieldName<S>): string
  /** 送信を試みたとき、まだ触れていない欄の問題も表示する。 */
  revealAll(): void
}

function problemOf(issue: z.core.$ZodIssue): FieldProblem {
  const problem = issue.code === 'custom' ? issue.params?.problem : undefined
  return (problem as FieldProblem | undefined) ?? { kind: 'invalid' }
}

/** 入力中の値を規則で検査し、送信する値と、欄ごとに表示する問題を返す。 */
export function useFormValidation<S extends FormSchema>(
  schema: S,
  values: Values<S>,
): FormValidation<S> {
  const id = useId()
  const [initial] = useState(values)
  const [touched, setTouched] = useState<ReadonlySet<string>>(() => new Set())
  const [revealed, setRevealed] = useState(false)

  const result = schema.safeParse(values)
  const problems = new Map<string, FieldProblem>()
  if (!result.success) {
    for (const issue of result.error.issues) {
      const name = String(issue.path[0])
      if (!problems.has(name)) problems.set(name, problemOf(issue))
    }
  }

  const isShown = (name: FieldName<S>) =>
    revealed || touched.has(name) || !Object.is(values[name], initial[name])
  const problem = (name: FieldName<S>) => (isShown(name) ? problems.get(name) : undefined)
  const errorId = (name: FieldName<S>) => `${id}-${name}-error`

  return {
    data: result.success ? result.data : undefined,
    problem,
    errorId,
    fieldProps: (name) => {
      const invalid = problem(name) !== undefined
      return {
        'aria-invalid': invalid ? true : undefined,
        'aria-describedby': invalid ? errorId(name) : undefined,
        onBlur: () =>
          setTouched((current) => (current.has(name) ? current : new Set(current).add(name))),
      }
    },
    revealAll: () => setRevealed(true),
  }
}

type TextFieldName<S extends FormSchema> = {
  [K in FieldName<S>]: Values<S>[K] extends string ? K : never
}[FieldName<S>]

export type Form<S extends FormSchema = FormSchema> = FormValidation<S> & {
  values: Values<S>
  setValue<K extends FieldName<S>>(name: K, value: Values<S>[K]): void
  /** 文字列の欄の入力要素に渡す、値、変更の処理、検証の属性。 */
  input(name: TextFieldName<S>): ReturnType<FormValidation<S>['fieldProps']> & {
    value: string
    onChange: (event: { target: { value: string } }) => void
  }
}

/** 入力中の値を保持し、`useFormValidation` で検査する。値を画面の状態として別に持たないフォームに使う。 */
export function useForm<S extends FormSchema>(schema: S, initial: Values<S>): Form<S> {
  const [values, setValues] = useState(initial)
  const validation = useFormValidation(schema, values)
  const setValue = <K extends FieldName<S>>(name: K, value: Values<S>[K]) =>
    setValues((current) => ({ ...current, [name]: value }))
  return {
    ...validation,
    values,
    setValue,
    input: (name) => ({
      ...validation.fieldProps(name),
      value: values[name] as string,
      onChange: (event) => setValue(name, event.target.value as Values<S>[typeof name]),
    }),
  }
}
