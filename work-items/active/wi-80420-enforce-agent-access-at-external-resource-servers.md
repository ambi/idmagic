---
status: pending
authors: [tn]
risk: high
created_at: 2026-10-30
priority: p1
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/authorization/features/check.md, requirement: REQ-AUTHORIZATION-004 }
  - { path: docs/modules/authorization/features/check.md, requirement: REQ-AUTHORIZATION-005 }
  - { path: docs/modules/authorization/features/model.md, requirement: REQ-AUTHORIZATION-010 }
  - { path: docs/modules/oauth2/features/authorization.md, requirement: REQ-OAUTH2-054 }
---

# 外部のリソースサーバーが機械認証でエージェントの操作を認可できるようにする

## 動機

IdMagic は Agent、RAR、委譲チェーン、ReBAC を持つが、関係判定の HTTP は管理者の診断用であり、API アクセストークンからも呼べない。
外部の MCP サーバーと業務 API は Go ユースケースを直接呼べないため、トークンの発行時に決めた制約を実行直前に確かめる接点がない。
AuthZEN の既存 HTTP アダプターは外部 PDP を利用する側であり、外部資源が IdMagic に判定を依頼する入口とは異なる。

## 対象範囲

- 登録済みリソースサーバーが、自身の資源への操作だけを機械認証で評価する公開契約を追加する。
- 提示トークンから検証済みの主体、行為者、テナント、audience、scope、RAR、失効を解決し、関係判定と合成する。
- 呼び出し元が本文で申告した subject や actor を認証結果の代わりに使わない。
- 許可の鮮度、キャッシュ、停止の伝播、評価不能時の拒否を契約にし、最小の参照 PEP と独立した MCP サーバーで確認する。
- [[wi-59-agent-governance-guardrails-audit-inventory]] は上限とポリシーを所有し、本項目はその判断を外部資源で執行する接点を所有する。

## 対象外

- 汎用 API gateway、WAF、プロンプト検査、全フレームワーク向け SDK。
- 管理用のモデル更新 API の機械公開、業務処理の成功や原子性の保証。
- 新しい認可言語、独自 Agent ID 規格、未実装ガードレールの先取り。

## 設計

要件の差分は、管理者限定の診断経路を保ちながら、登録した資源の評価専用経路を追加することである。
許可は対象の登録、検証済みトークン、現在の権限の論理積とし、失効や依存を判定できなければ業務 API を呼ばない。

境界の入力は、AccessControl の関係判定、OAuth2 の検証と audience、IdManagement の主体、SharedSignals の失効、既存のモジュール責務表とする。
D1/D2/D3/D6 に従い、判定は AccessControl、資格情報とトークンの検証は OAuth2、状態は各所有者に残す案を第一候補とする。
既存 admin API をそのまま開く案は、診断能力と資源所有権を混ぜるため採らない。
参照 PEP は別々に配布されるため D9 の公開 HTTP 契約だけで結び、サーバー内部型を import しない。
鍵方式追加時の波及は OAuth2 の検証アダプター、関係規則変更時の波及は AccessControl に閉じることを比較基準とする。
新しい依存が循環を作る場合は、利用側の最小ポートと組み立て地点による接続を比較し、実装前に採否を記録する。

主要な型の候補は `ResourceAccessRequest`、`VerifiedAccessContext`、`ResourceAccessDecision` とする。
操作候補は `EvaluateResourceAccess(ctx, authenticatedResource, token, operation, now)` とし、検証、現在状態の読み出し、監査を明示ポートで受ける。
関係と制約の合成はデータ上の計算に保つ。
AuthZEN の外部契約への適合範囲、資源サーバーの認証方式、DPoP の検証場所、実行と判定の時間差、許可応答の再利用期間は実装前に確定する。

移行は既存管理 API と別の入口を追加し、資源ごとに PEP を接続する。
ロールバックで保護を外さず、評価経路を止めると依存資源が拒否する状態へ戻す。
JWT のオフライン検証だけでは即時停止を保証できず、外部で既に確定した業務操作を取り消さないことを明記する。

## 計画

1. 既存要件との差分と外部の公開契約を定め、構築内容を変える問いを解決する。
2. 一つの外部 MCP 資源で、正常許可と停止後の拒否までの利用側を成立させる。
3. 共通の判定へ接続し、資源、テナント、actor の取り違えを拒否する。

## タスク

- [ ] T001 [Design] 認証、検証場所、鮮度、依存と責任分界を比較し、主要ユースケースを確定する。
- [ ] T002 [Spec] 既存の管理者限定規則と新しい評価規則を区別し、TypeSpec、機能仕様、脅威モデルを更新する。
- [ ] T003 [Acceptance] 公開入口から外部資源の許可、失効後の無作用、別資源への要求の拒否を観測する RED を作る。
- [ ] T004 [App] 検証済み文脈の解決と関係判定を既存ポートへ接続し、拒否理由と監査を実装する。
- [ ] T005 [PEP] 公開契約だけを使う最小参照 PEP と接続例を作り、評価不能時に業務処理を呼ばないようにする。
- [ ] T006 [Verify] テナント越え、audience 違い、偽の actor、期限、停止、キャッシュ、並行した権限変更を検証する。

## 検証

- `mise run check-spec`、`mise run check-boundaries`、`mise run check-published-api-compat`：契約、公開範囲、既存 API の互換性を確かめる。
- `mise run test-go-package -- ./accesscontrol/...` と `mise run test-go-package -- ./oauth2/...`：正常対照、誤った資源と主体、判定不能、停止を検証する。
- `mise run test-go-package -- ./shared/http/server_http`：正式な組み立てで参照 PEP からの要求と、拒否後に業務作用が発生しないことを確かめる。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。HTTP と DB を使う検証はサンドボックス外で実行する。
- 判定の単体テストと、外部の資源側で作用が起きないことの検査を独立した証拠にする。

## リスク

PDP を機械公開すると、主体や資源の申告を信じた場合に権限探索や混乱した代理人の入口になる。
資源サーバー自身の認証とトークンの audience を束縛し、検証された主体のみを使う。
許可の鮮度を曖昧にすると停止保証が成立しないため、キャッシュの上限と障害時の拒否を利用側も検証する。
