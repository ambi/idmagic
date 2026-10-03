---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-01
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-006 }
---

# テナントを解決していない文脈で default テナントへ落ちない

## 動機

テナントの解決は、どの経路にも一致しないリクエストを default テナントへフォールバックさせないと定める（REQ-TENANCY-006）。
しかし、リクエストの文脈からテナント ID を読む `tenancy.TenantID` は、文脈にテナントがないとき default テナントの ID を返す。
本番コードには、この関数の呼び出しが 252 箇所ある。

テナントを解決するミドルウェアを通らない経路や、解決の後に文脈を作り直す処理がこの関数を呼ぶと、その処理は default テナントのデータを読み書きする。
Tenancy の既存コードを書き起こしたときに見つけたが、観測できる境界での結果として規則に書くには、呼び出し元の経路を調べる必要がある。

## 対象範囲

- `tenancy.TenantID` の呼び出し元のうち、テナントを解決していない文脈で呼ばれ得るものを洗い出す。
- 文脈にテナントがないときに失敗させる形へ変え、default テナントへ暗黙に落ちる経路をなくす。
- 意図して default テナントを使う経路は、default テナントを明示して呼ぶ。

## 対象外

- テナントの解決の規則そのものの変更。

## 計画

1. 呼び出し元を、HTTP の経路、ジョブ、起動処理に分けて洗い出す。
2. 文脈にテナントがない呼び出しを検出するテストを先に書く。
3. 失敗させる形へ変え、default テナントを明示する経路を分ける。

## タスク

- [ ] T001 [Plan] 呼び出し元を洗い出す。
- [ ] T002 [Spec] 文脈にテナントがない処理の結果を規則に書く。
- [ ] T003 [App] 暗黙の default テナントをなくす。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check`
- `mise run verify`

## リスク

- 現在 default テナントへ落ちることで動いている処理が、失敗に変わる。
  洗い出しで意図した利用を先に明示する。
