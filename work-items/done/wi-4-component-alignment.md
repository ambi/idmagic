---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-06-14
---

# idmagic の package 構成と scl.yaml components の整合性を取る

## Motivation
SCL のモジュール (components) は DDD のサブドメイン分割であり、実装はその
境界に整合していることが RA の必須条件である。現状、Go 側は protocol-family
ベースで oauth2 / authentication / tenancy / administration の 4 パッケージに
分かれているが、SCL は Tenancy / Identity / ClientRegistry / Authorization /
Token / Trust の 6 モジュールに分かれており、両者は 1:1 に対応していない。

ユーザ判断: Go 側の protocol-family 分割は綺麗に保ち、SCL 側を Go パッケージ
境界に合わせて再編する。SCL に Go 特有概念は持ち込まない。

## Scope
- **scl**:
  - components.Identity を components.Authentication にリネームする (owns_models / owns_events / owns_interfaces / owns_permissions / owns_invariants / owns_objectives / depends_on をそのまま引き継ぐ)。
  - components.Authorization / Token / Trust / ClientRegistry の 4 つを components.OAuth2 単一に統合する。owns_* を union し、depends_on は 重複排除して Tenancy / Authentication への参照のみ残す。
  - 各セクションの annotations.component に書かれた旧名 4 つを OAuth2 に、 Identity を Authentication に一括置換する。
- **go**:
  - internal/administration/usecases/users.go を internal/authentication/usecases/admin_users.go に移動 (test 含む)。
  - internal/administration/usecases/clients.go を internal/oauth2/usecases/admin_clients.go に移動。
  - internal/administration/usecases/consents.go を internal/oauth2/usecases/admin_consents.go に移動。
  - internal/administration/ パッケージを削除。
  - internal/oauth2/ports/session_store.go を internal/authentication/ports/session_store.go に移動 (LoginSession は Authentication が所有するため)。
  - 上記移動に伴う import 文を全箇所更新する。

## Out of Scope
- internal/oauth2/ をさらに細かい Go パッケージに分割する大規模 refactor。
- Go 側コードのロジック変更や API 互換性に影響するリネーム。
- SCL の models / interfaces / permissions の追加・削除。

## Verification
- go build ./... が成功する。
- go vet ./... が無警告。
- go test ./... が全 pass。
- SCL の owns_* に列挙された全要素が models / interfaces 等で見つかる (coherence test SCLPermissionsCoverage 等)。
- rg -n "component: (Identity|Authorization|Token|Trust|ClientRegistry)" で旧 component 名が scl.yaml 内に残らないこと。
- find internal/administration が空 / 不在であること。

## Risk Notes
SCL 4 components の union は機械的だが、annotations.component の一括置換と
go の import 修正は広範。途中状態でコミットしないこと。

## Completion
- **Completed At**: 2026-06-14
- **Summary**:
  SCL components を Tenancy / Authentication / OAuth2 の 3 つに再編し、
  idmagic の package 構造と 1:1 に対応させた。Go 側は invented vocabulary の
  internal/administration/ を解体し、Authentication / OAuth2 配下に再配置した。
  LoginSession を扱う SessionStore port を SCL の所有関係に合わせて
  oauth2/ports から authentication/ports へ移動した。
- **Verification Results**:
  - `go build ./...`
    - result: ok
  - `go vet ./...`
    - result: ok
  - `go test ./...`
    - result: ok
  - `grep -rE "component: (Identity|Authorization|ClientRegistry|Token|Trust)\b" idmagic/spec/scl.yaml`
    - result: empty (旧名は残らない)
  - `ls idmagic/internal/administration`
    - result: not found
