---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-11-07
priority: p3
depends_on: [wi-77559-verify-jwt-svid-against-spiffe-rules]
change_kind: feature
affected_spec:
  - { path: docs/modules/workloadidentity/features/attestation-exchange.md, requirement: REQ-WORKLOADIDENTITY-001 }
  - { path: docs/modules/oauth2/standards.md, requirement: RFC8693-SUBJECT-TOKEN }
  - { path: docs/modules/oauth2/standards.md, requirement: RFC8705-CLIENT-AUTH }
  - { path: spec/modules/workloadidentity/models.tsp, symbol: IdMagic.Contract.WorkloadTrustBundle }
---

# X.509-SVID を mTLS で提示したワークロードを Agent として認証する

## 動機

WorkloadIdentity は JWT のアテステーションだけを受け付け、X.509-SVID（mTLS）は将来の拡張としている。
[設計判断](../../docs/modules/workloadidentity/decisions.md#外部のアテステーションを検証する-rp-に徹し長期のシークレットを配らない)は、JWT 以外のアテステーションが要る場合を見直しの条件に挙げている。

SPIFFE の基盤では、X.509-SVID が主要な形であり、JWT-SVID は再生の危険があるため補助として扱われることが多い。
X.509-SVID は TLS の鍵の所持を証明するので、JWT-SVID より再生に強い。
また、発行したアクセストークンを証明書に束縛（RFC 8705）すれば、盗まれたトークンの悪用も防げる。

## 対象範囲

- 信頼設定に、X.509 のトラストバンドル（ルート証明書の集合）を登録できるようにする。
- トークンエンドポイントで、クライアント証明書の X.509-SVID を検証する。URI SAN の SPIFFE ID がちょうど一つで、トラストドメインが信頼設定と一致することを求める。
- 検証した SPIFFE ID を関連付けのパターンと照合し、対応先の Agent を決める。
- 発行するアクセストークンを証明書のサムプリントに束縛するか（`cnf.x5t#S256`）を決める。

## 対象外

- IdMagic が X.509-SVID を発行すること。RP に徹する判断は変えない。
- リソースサーバー側での mTLS の強制。

## 設計

着手前に決める問いが多いため、ここでは問いを並べる。

| 問い | 候補 |
| --- | --- |
| 提示の経路 | (a) RFC 8705 の `tls_client_auth` 相当でトークンエンドポイントへ直接 mTLS する。(b) Token Exchange の `subject_token` に証明書を載せる独自の `subject_token_type` を使う |
| TLS の終端 | IdMagic が TLS を終端するか、前段のロードバランサーが終端して証明書をヘッダーで渡すか。後者ではヘッダーの信頼の境界を定める必要がある |
| 失効 | X.509-SVID は短命なので失効の確認をしない、という SPIFFE の前提を採るか |

(b) は標準の外になるため、(a) を推す。
TLS の終端は配備の構成に依存するため、[配備の設計](../../docs/design/README.md)と合わせて決める。
この作業はモジュール間の依存（`OAuth2` のクライアント認証から `WorkloadIdentity` への依存）を足すため、着手時に[境界を選ぶ判断手順](../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)の記録をここに書く。

## 計画

1. 上の問いを決め、境界の判断を記録する。
2. 仕様、TypeSpec、OAuth2 の標準の表を更新する。
3. 証明書の検証、Agent の特定、トークンの束縛の順に実装する。

## タスク

- [ ] T001 [Design] 提示の経路、TLS の終端、失効の扱いを決め、境界の判断を記録する。
- [ ] T002 [Spec] 仕様、TypeSpec、OAuth2 の標準の表を更新する。
- [ ] T003 [Acceptance] X.509-SVID でアクセストークンを得る経路を RED にする。
- [ ] T004 [App] 証明書の検証と Agent の特定を実装する。
- [ ] T005 [App] 証明書への束縛を実装する（T001 で採る場合）。
- [ ] T006 [UI] 信頼設定に X.509 のトラストバンドルを登録できるようにする。
- [ ] T007 [Verify] 検証を実行する。

## 検証

- `mise run test-go-package -- ./backend/workloadidentity/...` と `./backend/oauth2/...`。
- `mise run check-schema`。
- mTLS を張る HTTP の統合テスト（localhost の待受けを使うため、サンドボックスの外で実行する）。

## リスク

- 前段で TLS を終端する構成では、証明書を渡すヘッダーを偽造されると成りすましが起きる。信頼する送信元を設定で限り、既定では無効にする。
- 公開契約に新しい認証方式を足すため、後から取り除けない。
