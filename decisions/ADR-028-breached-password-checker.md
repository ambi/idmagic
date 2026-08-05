---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-028: 漏洩パスワード検査ポートと HIBP k-anonymity 採用

## コンテキスト

ADR-026 で「HIBP 等の外部漏洩データベース検査は `password_policy` とは別に
`BreachedPasswordChecker` port を経由する」と明文化したが、実装は ADR-027 と同時に
保留していた。本 ADR は port の存在意義・採用 adapter・失敗時挙動を定める。

外部漏洩データベース検査は、bundled common-password 辞書（offline / 即時）が拾えない
**過去に大規模流出に含まれた具体的なパスワード文字列** を弾くために必要である。
NIST SP 800-63B-4 §3.1.1.2 は「subscriber chooses a password, the verifier SHALL compare
the prospective secrets against a list that contains values known to be commonly-used,
expected, or compromised」と要求する。bundled 辞書は最小限の baseline、
外部知識は port に委ねる。

## 決定

`BreachedPasswordChecker` port (`isBreached(plain): Promise<boolean>`) を導入し、本番 adapter は
HIBP Range API を k-anonymity（SHA-1 先頭 5 文字のみ送信）で呼ぶ。デフォルト adapter は
`NoopBreachedPasswordChecker` とし、外部依存の無い in-memory 起動でも login / change-password が
動く可逆性を保つ。失敗時は fail-open（breached=false）とする — 外部サービス停止で
change-password 全体が止まるリスクを避け、漏洩検査を bundled 辞書・長さ・履歴と独立に片肺運用
できる追加レイヤーとして設計する。

現在の設計は [`backend/authentication/ARCHITECTURE.md`](../backend/authentication/ARCHITECTURE.md)
の Password lifecycle セクションにある。

## 影響

- 新 port `BreachedPasswordChecker`（`isBreached` 1 メソッド）。
- 新 adapter 2 種:
  - `NoopBreachedPasswordChecker`（既定 / memory）
  - `HibpBreachedPasswordChecker`（外部 API / opt-in）
- `password-policy.ts` に `breached` 違反語彙と `validatePasswordAsync` を追加。
  既存 `validatePassword`（同期）は seed / 純検査用に残す。
- `change-password.ts` の DI に `breachedPasswordChecker` を追加。
- `bootstrap/dependencies.ts` で `BREACHED_PASSWORD_CHECKER=hibp` のとき HIBP adapter、
  それ以外は Noop を返す（未指定が既定）。

## 却下した代替案

- **HIBP 結果の数値 (count) を usecase に持ち込む**: 閾値判断を usecase に
  漏らすと「いくつまでなら許すか」の policy 表現が分散する。port は二値、
  count 閾値判定は adapter 内に閉じる。

- **失敗時に fail-closed（変更を拒否）**: 外部サービス障害で IdP の
  change-password が利用不能になる。漏洩検査は補強レイヤであり、blocking 化
  すると可用性を犠牲にする。可用性側に倒し、検査失敗は監査ログで補完する。

- **bundled 辞書を巨大化して外部 API を不要にする**: ADR-026 で却下済み
  （配布サイズ・更新性で劣る）。HIBP 採用で本 ADR は閉じる。

- **plain password 全体を外部に送る**: k-anonymity プロトコル違反。
  HIBP Range API は prefix 5 文字送信が前提。
