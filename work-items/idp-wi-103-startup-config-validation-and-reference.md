---
id: idp-wi-103-startup-config-validation-and-reference
title: "起動時の設定を集約・検証し fail-fast させ、単一の設定リファレンスを生成する"
created_at: 2026-07-04
authors: ["tn"]
status: pending
risk: medium
---
# Motivation
現状の設定は bootstrap の各所で `envDefault` / `envInt` / `envDuration` を
直接呼んで読み、値の妥当性を集中検証しない。特に `envInt` / `envDuration` は
パース失敗や負値を「静かに fallback へ戻す」ため、`TRUSTED_FORWARDED_HOPS` や
リテンション間隔のような security/運用に効く値をタイポしても、警告なく既定値で
起動してしまう。本番でこれは、意図した閾値が実は効いていないという silent
misconfiguration を招く。設定項目の網羅一覧も存在せず、ARCHITECTURE.md も
「全環境変数一覧は置かない」としているため、運用者が正となる設定表を持てない。

12-factor と Kubernetes のコンポーネント設定検証（無効な設定は起動拒否）に倣い、
idmagic も設定を 1 つの型へ集約し、起動時に検証して不正なら fail-fast し、
型定義から機械生成した設定リファレンスを提供すべきである。ISSUER のような
必須値の欠落・不正 URL、相互矛盾する組み合わせ（postgres 指定なのに DSN 無し等）を
起動時に明確なエラーで止める。

# Scope
- **decision**: 新規 ADR: 設定を集約する Config 型の位置づけ、fail-fast の対象（必須欠落・型不正・ 範囲外・相互矛盾）と、後方互換のために warning に留める範囲を定義する。 secret は値をログに出さない方針を明記する。
- **go**: env 由来設定を単一の Config 構造体へ集約してパース・検証する層を bootstrap に追加する。 検証失敗は Run() の起動前に集約エラーで返し、部分起動させない。, `envInt` / `envDuration` の「不正値を黙って fallback」を、少なくとも security/運用に 効く項目では明示エラー化する。範囲・必須・相互依存（persistence=postgres なら DSN 必須等）を検証する。, 検証済み Config を各 assemble / handler へ渡し、散在した os.Getenv 直参照を減らす。, secret（DSN・SMTP 資格情報・API キー等）は検証エラーやログに値を出さない。
- **documentation**: Config 型定義から設定リファレンス（キー名・型・既定値・必須・意味）を生成し、 README から参照できるようにする。手書き一覧の二重管理を避ける。

# Out of Scope
- 動的な設定ホットリロード。
- 外部設定サービス（Consul / etcd 等）連携。
- 既存の環境変数キー名の一斉改名（互換維持を優先）。

# Verification
- [object Object]
- [object Object]
- 手動: 必須値欠落・不正 URL・矛盾する組み合わせで起動させ、明確なエラーで停止し 部分起動しないことを確認する。
- 手動: 生成された設定リファレンスが実 Config 型と一致することを確認する。

# Risk Notes
黙って fallback していた挙動を fail-fast に変えると、既存のゆるい設定で
動いていた環境が起動できなくなり得る。security に効く項目から段階導入し、
非致命項目は当面 warning に留めるなど、移行の破壊範囲を ADR で線引きする。
