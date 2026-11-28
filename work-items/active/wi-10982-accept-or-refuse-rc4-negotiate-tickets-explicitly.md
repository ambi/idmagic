---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-11-28
priority: p1
group: authentication
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/authentication/features/negotiate.md, requirement: REQ-AUTHENTICATION-042 }
---

# RC4 の Kerberos のサービスチケットを、PAC の解析の失敗ではなく明示の規則で扱う

## 動機

2026-11-28 に [[wi-21740-decide-recordings-for-requests-idmagic-receives]] で、隔離した Samba AD から macOS の kgetcred が取ったサービスチケットは RC4-HMAC（etype 23）だった（`backend/authentication/internal/negotiate/protocol_kerberos/testdata/chrome-negotiate.json`）。
AD はサービスのアカウントの `msDS-SupportedEncryptionTypes` が未設定のとき、サービスチケットを RC4 で出すことがある。
記録のときに束縛の検査を外して gokrb5 の受理を動かすと、`process service ticket PAC: ... PAC is malformed: checksum type 4294967158 exceeds int32 range` で失敗した。
RC4 の PAC の署名のチェックサムの種類は HMAC-MD5（-138）であり、gokrb5 がそれを符号なしの値として読んで範囲の検査で拒否していると考えられる。
既存の `TestVerifierAgainstSambaAD` は gokrb5 のクライアントでチケットを取る。
2026-11-29 に `mise run test-kerberos-ad` を実行すると、このテストは `main` の時点で `negotiate: SPNEGO or channel binding validation failed` で失敗した（infra/test/kerberos/start.sh を変える前の版でも同じである）。
このテストは CI で実行されておらず、いつから失敗しているかは分からない。
原因が RC4 の PAC か別の理由かは、まだ確かめていない。着手時に最初に確かめる。

その結果、AES を有効にしていない AD のサービスのアカウントでは、すべての Negotiate のログインが「SPNEGO or channel binding validation failed」という理由の分からない拒否になる。
手順書（docs/runbooks/kerberos-setup.md）は暗号方式に触れていない。
利用者と管理者が誤った結果を受け取るので、グループの基準の p3 ではなく p1 とする。

## 対象範囲

- RC4 のチケットを受け付けるか拒否するかを決め、negotiate.md と手順書に書く。RC4 は RFC 8429 が非推奨とする。
- 拒否するなら、拒否の理由を監査と `check-kerberos-config` で区別できるようにし、手順書にサービスのアカウントで AES を有効にする手順を書く。
- 受け付けるなら、gokrb5 の PAC の解析を直す（このリポジトリが使う `otuschhoff/gokrb5` への修正か、取り込みの前の回避）。
- 検査用の Samba AD（infra/test/kerberos）で、RC4 と AES のサービスのアカウントの両方を試す。
- `TestVerifierAgainstSambaAD` の失敗を直し、`test-kerberos-ad` を CI で実行するかを決める。

## 対象外

- チャネルバインディングを送らないクライアントの扱い。[[wi-55734-decide-negotiate-for-clients-without-channel-bindings]] が扱う。

## 設計

着手時に、gokrb5 の PAC の署名のチェックサムの種類の扱いを読み、範囲の検査が仕様（MS-PAC の PAC_SIGNATURE_DATA の SignatureType は 32 ビットの符号付き整数）に反しているかを確かめる。

## 計画

1. 方針を決めて仕様と手順書を更新する。
2. Samba AD の RC4 と AES のチケットで RED を観測し、直す。

## タスク

- [ ] T001 [Spec] RC4 の扱いを仕様と手順書に書く。
- [ ] T002 [Acceptance] RC4 のチケットで RED を確認する。
- [ ] T003 [App] 扱いを実装する。
- [ ] T004 [Verify] 差分から選んだ検査を実行する。

## 検証

- `mise run test-kerberos-ad`、`mise run test-go-package -- ./backend/authentication/internal/negotiate/protocol_kerberos`。

## リスク

RC4 を受け付けると弱い暗号のチケットを信頼する。
受け付ける場合も既定は拒否にし、テナントの明示の設定に限る。
