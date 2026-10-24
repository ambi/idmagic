---
status: pending
authors: [tn]
risk: medium
created_at: 2026-10-24
priority: p3
depends_on: [wi-69206-run-a-directory-connector-that-reads-ad-and-ldap]
change_kind: feature
affected_spec:
  - { path: docs/modules/directory-connector/features/runtime.md, requirement: REQ-DIRECTORYCONNECTOR-005 }
---

# Directory Connector を Windows のサービスとして登録し、サービスマネージャーの停止で終える

## 動機

AD を持つ閉域では、Connector を Windows Server に置くことが多い。
今の Connector は `run` を常駐させるだけで、Windows ではタスク スケジューラの起動時のタスクとして動かすよう手順書が案内している。
これでは停止の合図を受け取れず、サービスの再起動の方針や状態の確認を Windows の標準の道具で扱えない。
Okta の AD Agent は Windows のサービスとして導入され、サービスとして停止と再起動ができる。

## 対象範囲

- `install-service` と `uninstall-service` のサブコマンドで、実行のアカウントと設定ファイルを指定してサービスを登録、解除する。
- サービスとして起動したとき、サービスマネージャーの停止の要求を停止の合図として扱い、REQ-DIRECTORYCONNECTOR-005 の常駐の停止と同じく終える。
- ログを Windows のイベントログか、設定したファイルへ出す。
- 手順書の Windows の起動の手順を書き換える。

## 対象外

- MSI などのインストーラーと、自動更新。
- Linux の systemd のユニットファイルの同梱。

## 設計

### 未解決の問い

- `golang.org/x/sys/windows/svc`（既存の依存の `golang.org/x/sys` に含まれる）を使うか。
- ログの出力先をイベントログとファイルのどちらにするか。

## 計画

1. 未解決の問いを決め、仕様に要件を足す。
2. Windows の CI ランナーでサービスの登録と停止を確かめる方法を決めてから実装する。

## タスク

- [ ] T001 [Plan] 未解決の問いと検証の方法を決める。
- [ ] T002 [Spec] 要件を足す。
- [ ] T003 [App] サービスの登録、解除、停止を実装する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- Windows の環境で、サービスの登録、起動、停止、解除を確かめる。

## リスク

- Windows の CI ランナーがないと、サービスの振る舞いを自動で確かめられない。
