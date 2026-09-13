---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-13
priority: p3
depends_on: []
change_kind: docs
spec_impact: { kind: none, reason: "本項目は、宣言済みの拒否と現在の実装が異なる境界またはエラー種別を使う二つの食い違いについて、どちらを正とするか決める。規範または実装を変更する作業は、判断後に仕様先行で進める。" }
---

# 異なる境界で実施している OAuth2 の拒否を仕様と一致させる

## Motivation

`EX-OAUTH2-026-01` は、public クライアントへ `client_credentials` を指定した登録要求を `InvalidRequestError` で拒否すると宣言している。
現在の実装は登録を受理し、`/token` で `unauthorized_client` を返す。

`EX-OAUTH2-035-02` は、別テナントのクライアント参照を `InvalidRequestError` で拒否すると宣言している。
現在の実装は対象の存在を隠すため、404 Not Found を返す。

どちらも拒否は成立しているが、拒否する境界または利用者が観測するエラー種別が規範と一致しない。
WI-559 が測定結果を台帳へ残したため、本項目が判断と解消を引き取る。

## Scope

- `EX-OAUTH2-026-01` について、public クライアントを登録時に拒否するか、トークンエンドポイントで拒否する現在の境界を規範へ反映するかを決める。
- `EX-OAUTH2-035-02` について、テナントを越えた参照で対象の存在を隠すか、宣言済みの `InvalidRequestError` を返すかを決める。
- 決定に従って仕様を先に更新し、必要なら実装を変更する。
- 各具体例を名指しするテストで、エラー種別と拒否が防いだ効果を固定してから被覆台帳から外す。

## Out of Scope

- `client_credentials` グラントの成功経路。WI-559 のテストが固定している。
- OAuth2 管理 API 全体のテナント境界の再設計。
- 本項目が扱う二つ以外の被覆台帳項目。

## Design

`EX-OAUTH2-026-01` では、登録契約とトークンエンドポイントの責務を分けて判断する。
登録時の拒否を採る場合は、public クライアントが別のグラントを利用できる可能性を失わない入力条件を定める。
現在の境界を採る場合は、登録自体ではなくグラント利用を拒否することが読者へ伝わるように具体例を直す。

`EX-OAUTH2-035-02` では、対象の存在を別テナントへ漏らさない性質と、具体例が名指すエラー種別を同時に評価する。
404 Not Found を採る場合は非開示の理由を正準文書へ残し、`InvalidRequestError` を採る場合は情報開示が増えないことをテストで示す。

二つの判断を同じエラー名へ揃えることは目的にしない。
呼び出し境界と保護する情報が異なるため、それぞれの責務から結論を出す。

## Plan

1. TypeSpec、OAuth2 シナリオ、登録処理、管理 API のテナント解決を読み、現在の契約と非開示方針を確認する。
2. 各具体例について規範と実装のどちらを変更するか決め、仕様を先に更新する。
3. 観測可能な HTTP 境界で RED を確認し、必要な実装変更とテストを行う。
4. 被覆台帳から二件を外し、標準検証を通す。

## Tasks

- [ ] T001 [Spec] `EX-OAUTH2-026-01` と `EX-OAUTH2-035-02` の拒否境界とエラー種別を決め、仕様を更新する。
- [ ] T002 [Acceptance] 各具体例の現在の食い違いを HTTP テストで RED として固定する。
- [ ] T003 [App] 決定が実装変更を求める場合は、単体 RED から実装してリファクタリングする。
- [ ] T004 [Tests] エラー種別と拒否が防いだ効果を検査するテストを追加し、被覆台帳から二件を外す。
- [ ] T005 [Verify] `mise run verify` を通す。

## Verification

- `mise run test-go-package -- ./backend/shared/http/server_http`
- `mise run test-go-package -- ./backend/oauth2/handlers_http`
- `mise run check-spec`
- `mise run check-work-items`
- `mise run verify`

## Risk Notes

- 登録時の拒否へ寄せると、public クライアントが利用できる別のグラントまで登録不能にするおそれがある。登録要求のどの組み合わせを拒否するかを先に仕様で限定する。
- テナントを越えた参照のエラーを変えると、対象の存在を推測できる差が生じるおそれがある。状態行だけでなく応答本文と対象テナントの保存状態も検査する。
