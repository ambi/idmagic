# Feature: WS-Trust の能動的 STS の例

## Rule: REQ-WSFEDERATION-004 妥当な WS-Trust Issue は RSTR を返す

### Example: EX-WSFEDERATION-004-01 通常経路

- Given UsernameToken、MessageID、Timestamp、To、Action、RequestType、KeyType、AppliesTo が有効である
- When WS-Trust Issue の RST を受信する
- Then UsernameToken と RST の必須要素をすべて検証する
- Then RSTR を返す

### Scenario Outline: 条件ごとの結果

- Given UsernameToken、MessageID、Timestamp、To、Action、RequestType、KeyType、AppliesTo が有効である
- When WS-Trust Issue の RST を受信する
- Then <result>
- Then <result_2>

#### Examples:

  | example_id | result | result_2 |
  | --- | --- | --- |
  | EX-WSFEDERATION-004-02 | MessageID が Assertion の有効期間内に再利用されている | WsTrustTokenRejected を発行してプロトコルエラーを返す |
  | EX-WSFEDERATION-004-03 | UsernameToken の資格情報が不正である | 401 で拒否しトークンを発行しない |

## Rule: REQ-WSFEDERATION-005 不正なエンベロープの WS-Trust Issue は拒否する

### Example: EX-WSFEDERATION-005-01 通常経路

- Given RST の To、MessageID、AppliesTo、Action、RequestType、KeyType のいずれかが不正である
- When 不正な RST を受信する
- Then WsTrustTokenRejected を発行し、400 または 401 を返す
