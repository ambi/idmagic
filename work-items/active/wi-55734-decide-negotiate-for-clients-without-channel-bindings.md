---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-11-28
priority: p2
group: authentication
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/authentication/features/negotiate.md, requirement: REQ-AUTHENTICATION-042 }
---

# チャネルバインディングを送らない Kerberos のクライアントの扱いを決める

## 動機

IdMagic の Negotiate は `tls-server-end-point` のチャネルバインディングを必須とする（negotiate.md のセキュリティ上の考慮）。
手順書（docs/runbooks/kerberos-setup.md）は「ブラウザーのバージョンと OS により channel binding の対応が異なる。対象端末で確認する」と書くだけで、どのクライアントが使えるかを示していない。

2026-11-28 に [[wi-21740-decide-recordings-for-requests-idmagic-receives]] で、隔離した Samba AD の利用者 alice のチケットを macOS の kinit で取り、手元の Google Chrome 158（macOS の GSS.framework）が Negotiate の要求に送ったトークンを記録した（`backend/authentication/internal/negotiate/protocol_kerberos/testdata/chrome-negotiate.json`、`mise run record-negotiate`）。
Chrome の net-log では Chrome が `tls-server-end-point` の束縛を計算していたが、送られた AP-REQ の authenticator のチェックサム（RFC 4121 §4.1.1）の Bnd は 16 バイトすべてが 0 だった。
macOS の GSS.framework が束縛をトークンへ入れていないと考えられる。
記録のときに IdMagic の Verifier と同じ設定で gokrb5 の受理を動かすと、`GSSAPI channel bindings do not match` で拒否された。
したがって、macOS の Chrome の利用者は今の設定では Negotiate でサインインできない。

## 対象範囲

- 束縛を送らないクライアントを拒否し続けるか、テナントの設定で受け付けるかを決める。
- 決めた扱いを negotiate.md の要件と、手順書の対応するクライアントの一覧に書く。拒否し続けるなら、拒否の理由が利用者と管理者に分かる形（監査の理由、手順書の既知の制限）にする。
- 受け付けるなら、受け付ける条件（テナントの明示の設定、TLS の終端の位置）と、失う保護（中間者による資格情報の中継）を脅威モデルに書く。
- `TestChromeNegotiate_TokenCarriesNoChannelBindings` を、決めた期待値で再生する。

## 対象外

- Windows の SSPI（Edge、Chrome on Windows）と Firefox のトークンの記録。手元に Windows の端末がなく、記録の取り方から決める別の作業である。
- RC4 のチケットの PAC。[[wi-10982-accept-or-refuse-rc4-negotiate-tickets-explicitly]] が扱う。

## 設計

着手時に、RFC 5929 と Microsoft の拡張保護（EPA）の「許可」の水準の意味、macOS の GSS.framework の束縛の扱いの一次資料を読み、選択肢を比べる。
束縛の検査は gokrb5 の `ExtendedProtection` の水準で切り替えられる（`Disabled`、`Allowed`、`Required`）。

## 計画

1. 一次資料を読み、選択肢と脅威を設計に書く。
2. `spec-change` で要件と手順書を更新する。
3. 記録の再生で期待値を確かめる。

## タスク

- [ ] T001 [Design] 選択肢と失う保護を比べる。
- [ ] T002 [Spec] 要件と手順書を更新する。
- [ ] T003 [Acceptance] 記録の再生で期待値を確かめる。
- [ ] T004 [Verify] 差分から選んだ検査を実行する。

## 検証

- `mise run test-go-package -- ./backend/authentication/internal/negotiate/protocol_kerberos`、`mise run record-negotiate`（手元）。

## リスク

束縛を任意にすると、TLS を終端する中間者がチケットを中継できる。
受け付けるなら、テナントの明示の選択に限り、既定は今の拒否を保つ。
