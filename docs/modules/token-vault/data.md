# TokenVault のデータ

この文書は、TokenVault の Aggregate を扱う。

## 集約

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `Connector` | エンドポイント、クライアントの資格情報、許可するスコープ、実行の規則、状態 | テナントに属する |
| `Connection` | 暗号化したアクセストークンと更新トークン、有効期限、付与スコープ、状態、`token_version`、更新のリース | `Connector` と User を一つずつ参照する。Connector の削除か User の削除で消える |
| `ConnectionAttempt` | state のハッシュ、暗号化した PKCE の検証子、要求スコープ、期限 | `Connector` と User を参照する。コールバックで一度だけ消費する |
