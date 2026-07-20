---
status: accepted
authors: [tn]
created_at: 2026-07-20
---

# ADR-131: OAuth2 HTTP protocol adapter は context root で共有する

## コンテキスト

OAuth2 の domain、port、usecase、persistence adapter を client / consent /
authorization / token / device の feature 垂直スライスへ再配置する過程で、HTTP handler は
単一 request 内で複数 feature と authentication context を横断することが確認された。
特に authorize flow は login、MFA、consent、authorization code を一つの transaction として
扱い、token endpoint は grant type によって authorization / token / device を dispatch する。

HTTP handler を feature ごとに分けるには、共有 `Deps` の leaf package 化だけでなく、
transaction 型、validation、client authentication、OAuth error mapping、login throttle を
再公開して feature 間を横断させる必要がある。これは物理配置変更を越えて protocol
orchestration の境界と wire contract を再設計する変更になる。

## 決定

feature 固有の domain、port、usecase、persistence adapter は feature 配下へ置く。一方、
OAuth2/OIDC protocol endpoint の HTTP handler、route 登録、wire validation、client
authentication、error mapping は `backend/oauth2/adapters/http` の context root shared
adapter に維持する。root HTTP adapter は feature usecase を named import して dispatch する。

共有 domain / port / persistence package の互換 facade は、既存 composition root と外部
context の公開 import を維持するため root に残す。現在の module path と依存方向は
`ARCHITECTURE.md` の OAuth2 module 群を正本とする。

## 却下した代替案

- HTTP handler を強制的に 5 feature へ分割する案: authorize と token の protocol
  orchestration を複数 package に分散し、共有 HTTP 型を公開 API 化する必要があるため採らない。
- `Deps` を feature ごとに分割する案: authorize handler が authentication、consent、client、
  authorization の依存を横断するため同じ port を複数 feature に重複定義することになる。
- root HTTP adapter に feature usecase を複製する案: 振る舞いの正本を二重化するため採らない。

## 影響

- SCL の interface、scenario、wire contract、context map は変更しない。
- wi-256 の当初 Scope にあった「HTTP handler を含む全 adapter の feature 配置」は、
  persistence adapter の feature 配置と root shared HTTP adapter に狭められる。
- 未対応範囲は HTTP handler 自体の feature ディレクトリ移動であり、完了報告で明示する。
- 将来 HTTP orchestration の境界を再設計する場合は、本 ADR を置き換えてから handler を移す。
