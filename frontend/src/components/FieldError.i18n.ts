import { defineDictionary } from '../lib/i18n'

export const fieldErrorDictionary = defineDictionary(
  {
    required: 'この項目を入力してください。',
    tooLong: '{max} 文字以内で入力してください。',
    tooMany: '{max} 件以内にしてください。',
    invalidCidr:
      '「{entry}」は CIDR ではありません。192.0.2.0/24 のように、アドレスと接頭辞長を入力してください。',
    positiveInteger: '1 以上の整数を入力してください。',
    nonNegativeInteger: '0 以上の整数を入力してください。',
    integerRange: '{min} 以上 {max} 以下の整数を入力してください。',
    dateTime: '2026-12-31T23:59:59Z のように、RFC 3339 の日時を入力してください。',
    invalidUrl: 'https://app.example.com のように、スキームとホストを含む URL を入力してください。',
    invalidUrlEntry:
      '「{entry}」は URL ではありません。https://app.example.com のように、スキームとホストを含めてください。',
    httpsUrl: 'https:// で始まり、ユーザー名と # 以降を含まない URL を入力してください。',
    httpsUrlEntry:
      '「{entry}」は使えません。https:// で始まり、ユーザー名と # 以降を含まない URL を入力してください。',
    resourceUrl: 'https://mcp.example.com/ のように、# 以降を含まない URL を入力してください。',
    invalidScope:
      '「{entry}」はスコープに使えません。空白で区切り、ASCII の記号と英数字だけを使ってください（" と \\ は使えません）。',
    invalidPem: '-----BEGIN CERTIFICATE----- で始まる PEM 形式の証明書を貼り付けてください。',
    invalidJson: 'JSON として読める値を入力してください。',
    invalid: '入力内容を確認してください。',
    characterCount: '{count} / {max}',
    characterCountNear: '残り {count} 文字です。',
    characterCountOver: '上限を {count} 文字超えています。',
  },
  {
    required: 'Enter a value for this field.',
    tooLong: 'Enter {max} characters or fewer.',
    tooMany: 'Enter {max} items or fewer.',
    invalidCidr:
      '"{entry}" is not a CIDR. Enter an address and a prefix length, such as 192.0.2.0/24.',
    positiveInteger: 'Enter a whole number of 1 or more.',
    nonNegativeInteger: 'Enter a whole number of 0 or more.',
    integerRange: 'Enter a whole number from {min} to {max}.',
    dateTime: 'Enter an RFC 3339 date and time, such as 2026-12-31T23:59:59Z.',
    invalidUrl: 'Enter a URL with a scheme and a host, such as https://app.example.com.',
    invalidUrlEntry:
      '"{entry}" is not a URL. Include a scheme and a host, such as https://app.example.com.',
    httpsUrl: 'Enter a URL that starts with https:// and has no user name or # fragment.',
    httpsUrlEntry:
      '"{entry}" cannot be used. Enter a URL that starts with https:// and has no user name or # fragment.',
    resourceUrl: 'Enter a URL without a # fragment, such as https://mcp.example.com/.',
    invalidScope:
      '"{entry}" cannot be used as a scope. Separate scopes with spaces and use only ASCII letters, digits, and symbols other than " and \\.',
    invalidPem: 'Paste a PEM certificate that starts with -----BEGIN CERTIFICATE-----.',
    invalidJson: 'Enter a value that parses as JSON.',
    invalid: 'Check this value.',
    characterCount: '{count} / {max}',
    characterCountNear: '{count} characters left.',
    characterCountOver: '{count} characters over the limit.',
  },
)
