# SigningKeys State Transitions

## SigningKeyLifecycle

署名鍵は `Active` からローテーションによって `Verifying` へ遷移し、`Retire` で JWKS から除外した後、`Archive` で監査保管に移す。公開鍵を重複して掲載する期間には `SigningKeyMinJwksOverlap` を適用する。

| State | Kind | Meaning |
|---|---|---|
| Active | initial | 新しいメッセージへの署名に使う。テナント、用途、スコープの組ごとに 1 本だけ存在する |
| Verifying | — | 署名には使わないが、JWKS とフェデレーションメタデータに載せて検証に使う |
| Retired | — | JWKS から除外した。公開しない |
| Archived | terminal | 監査保管。退役した鍵で署名された監査トークンを検証できるよう鍵素材を保持する |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | SigningKeyRotated | — | Verifying |  |
| Verifying | SigningKeyRetired | — | Retired |  |
| Retired | SigningKeyArchived | — | Archived |  |
