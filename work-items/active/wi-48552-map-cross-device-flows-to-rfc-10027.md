---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-11-17
priority: p2
group: federation-protocols
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/standards.md, requirement: RFC8628-DEVICE-AUTHORIZATION }
  - { path: docs/modules/oauth2/standards.md, requirement: CIBA-CORE-BACKCHANNEL-REQUEST }
  - { path: docs/modules/oauth2/standards.md, requirement: CIBA-CORE-BINDING-MESSAGE }
  - { path: docs/modules/oauth2/features/device.md, requirement: REQ-OAUTH2-027 }
  - { path: docs/modules/oauth2/features/approval.md, requirement: REQ-OAUTH2-043 }
---

# 別端末の認可要求とデバイスフローを RFC 10027 に対応付ける

## 動機

[RFC 10027](https://www.rfc-editor.org/rfc/rfc10027)（BCP 247、2026年8月）は、デバイス認可グラント、CIBA など別の端末で承認するフローの、同意のフィッシングとセッションのフィッシングへの対策を定めた。
多要素認証を使っていても成立する攻撃であり、実装前のリスク評価を MUST とする。

Agent はデバイスフローと CIBA で利用者の承認を得ることが多く、攻撃者が Agent を装って利用者に承認させる経路になる。
[Poppy 1.0](https://personalagentprotocol.org/docs/1.0/spec) §4.2 と §4.3 も RFC 10027 への準拠を要求する。
IdMagic の標準表にはこの BCP の行がなく、既存の対策との対応を示せない。
既存のフローの防御の穴を確かめる作業なので、群の基準の p3 より上げて p2 とする。

## 対象範囲

- RFC 10027 の推奨を、デバイスフロー、CIBA、Poppy の別端末の認可要求ごとに既存の規範とテストへ対応付けた適合表と、標準表の行。
- リスク評価の記録。
- 適合表で見つかった不足の実装。候補は、承認画面での要求元のクライアントと Agent、要求の地点や時刻の表示、要求の有効期間の短縮、別端末で開かれた認可要求の検出である。

## 対象外

- セッションの別端末への移送（cross-device session transfer）。IdMagic は提供していない。
- 近接の証明（BLE など）の実装。

## 設計

規則を複製せず、既存の要件で満たすものは参照と追加の証拠で示す。
不足が見つかった場合だけ新しい要件にする。
対策の選択は、RFC 10027 の脅威と緩和策の表を基準にし、利用者の手間を増やす対策（確認コードの入力など）は、リスク評価で必要と判断したフローに限る。

## 計画

1. RFC 10027 を読み、適合表とリスク評価を作る。
2. 不足を仕様へ反映し、受け入れ試験の RED を確認してから実装する。

## タスク

- [ ] T001 [Spec] 適合表、リスク評価、標準表の行を書く。
- [ ] T002 [Acceptance] 不足と判断した対策の RED を正式な HTTP 入口で作る。
- [ ] T003 [App] 不足を実装する。
- [ ] T004 [Verify] 検証する。

## 検証

- `mise run test-go-package -- ./backend/oauth2/internal/handlers_http`：デバイスフローと CIBA の契約。
- 承認画面を変えた場合は対象の `test-ui-e2e-file`。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>`。

## リスク

適合表を作るだけで対策済みと表示すると、実在の攻撃への耐性を過大に主張する。
緩和策ごとに、残る脅威を標準表の `partial` と説明に残す。
