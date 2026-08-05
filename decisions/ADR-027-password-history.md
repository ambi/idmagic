---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-027: パスワード履歴の再利用禁止

## コンテキスト

ADR-026 で「パスワード履歴の再利用禁止は `PasswordHistoryRepository` port と
change-password エンドポイントが揃った時点で別 ADR と共に追加する」と保留した。
本 ADR は `/api/auth/change_password` エンドポイントの導入に合わせて履歴ポリシーを
定義する。

履歴件数・保存形式・カスケード削除・テナント別カスタマイズの取り扱いは
将来の管理基盤（Phase 4 — RBAC / マルチテナンシー）に影響する設計判断のため、
spec↔impl drift を防ぐためここで明文化する。

## 決定

直近 5 件のパスワードハッシュとの一致を拒否する (`history_depth=5`)。5 は NIST SP 800-63B-4 の
禁止項目に該当しない範囲で、OWASP ASVS v4.0.3 §2.1.10 の "previously-used password" 検知要求と
整合する最小値であり、これ以上深くしてもユーザーの不便さに対する効果は逓減する。履歴は
`password_hash` と同じ Argon2id PHC エンコードのまま保存し、追加の暗号化は行わない — 本体と同じ
攻撃耐性を持つため、別建て暗号化しても閾値が下がらない。履歴は registration でも書くが、チェックは
change-password でのみ行う（初回登録時は照合対象がゼロ件のため）。

現在の設計は [`backend/authentication/ARCHITECTURE.md`](../backend/authentication/ARCHITECTURE.md)
の Password lifecycle セクションにある。

## 影響

- 新 port `PasswordHistoryRepository`（`add` / `recent` の 2 メソッド）。
- 新 use case `change-password.ts`（現パス verify → policy 検証 → 履歴照合 →
  hash → 保存 → 履歴追加 → `PasswordChanged` emit）。
- 新 HTTP endpoint `POST /api/auth/change_password`（セッション cookie 必須、
  CSRF 必須、JSON）。
- 新マイグレーション `password_histories(sub, encoded, created_at)` +
  `(sub, created_at DESC)` インデックス + `ON DELETE CASCADE`。
- 新 SPA 画面（change-password）。`/change_password` ルート、ja/en i18n。
- `bootstrap/seed.ts` は初期 hash を history に 1 件積む。

## 却下した代替案

- **深さを 12 / 24 等の大きな値にする**: NIST SP 800-63B-4 §3.1.1.2 は履歴禁止
  そのものを推奨も非推奨もしていないが、深さを増やすほど「直前のパスワードに
  +1 する」等の予測可能な書き換えを誘発する。OWASP ASVS の要件 (「previously-used
  passwords を検知できる」) は深さに下限を置いていない。5 は最小要件を満たす。
- **平文比較や独立ソルトでの再ハッシュ**: 既存 `password_hash` と異なる
  攻撃耐性を持ち込むことになり、二重管理になる。PHC エンコードをそのまま
  積むのが最も surgical。
- **履歴を user 行に JSON 配列として埋め込む**: 行の肥大・並列更新時の競合・
  カスケード削除の表現力に劣る。専用テーブルが妥当。
- **登録時に履歴を積まない**: 登録直後の change-password で初期パスへ戻せて
  しまう。1 件だけ積めば最小コストで埋まる穴。
