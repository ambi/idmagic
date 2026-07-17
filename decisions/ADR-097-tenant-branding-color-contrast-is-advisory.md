---
status: accepted
authors: ["tn"]
created_at: 2026-07-12
supersedes: [ADR-096]  # 決定 3（コントラスト比の扱い）のみ
---

# ADR-097: テナントブランディングの色コントラストは保存制約にしない

## コンテキスト

ADR-096 は hosted UI の可読性を守るため、ブランド色が既定背景に対して WCAG AA 4.5:1 を満たすことを保存時に要求した。しかし管理画面は判定基準や結果を表示しておらず、有効なブランドカラーを利用者が設定できないことがある。配色の責任はテナント自身が担う。

## 決定

primary_color と accent_color は `#rrggbb` 形式だけを保存制約とし、コントラスト比では拒否しない。管理画面も形式エラーだけを表示する。hosted UI への注入は従来どおり専用 CSS custom property に限定する。

この決定は ADR-096 の決定 3 にある保存時コントラスト拒否を置き換える。SCL の `models.TenantBranding`、`interfaces.UpdateTenantBranding`、`invariants.TenantBrandingSafeTokens`、scenarios、および `user_experience.screens.AdminSettings` に反映する。

## 却下した代替案

- コントラスト比を必須のまま、管理画面に判定表示を追加する: テナント固有の正しいブランドカラーを設定できない制約を解消しない。
- 色をすべて任意の CSS として受け入れる: XSS やレイアウト破壊の入力面を広げるため採らない。

## 影響

- 低コントラストの `#rrggbb` 値を API・永続化・hosted UI が保持・表示できる。
- 色の可読性低下はテナントが負う。入力形式、リンクの https 制約、テキスト長、画像形式・サイズ・配信保護は維持する。
