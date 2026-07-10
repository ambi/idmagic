---
status: accepted
authors: [tn]
created_at: 2026-07-11
---

# ADR-092: アプリケーションの実装ルートを backend と frontend に分ける

## コンテキスト
idmagic は外部に再利用させる Go ライブラリではなく、`idmagic` と
`idmagic-relay` を提供する単一モジュールのアプリケーションである。トップレベルの
`internal/` は Go compiler による module 外 import の拒否を提供するが、この構成では
実質的な保護境界になっていない。一方、`ui/` との命名は非対称で、他言語の開発者に
各成果物の役割が伝わりにくい。

## 決定
SCL の規範振る舞い、公開 API、HTTP route、DB schema は変更しない。

1. トップレベルの `internal/` を `backend/` に改名する。
2. `cmd/` を `backend/cmd/` に移し、Go の実装と実行 entry point を `backend/` に集約する。
3. `ui/` を `frontend/` に改名する。
4. Go module path は `github.com/ambi/idmagic` のままとし、Go import path の
   `/internal/` セグメントだけを `/backend/` に変更する。
5. ADR-068 の項目 4 と ADR-070 の `internal/` を前提とする記述は、この決定で置き換える。

## 却下した代替案
- `src/`: 一般的すぎ、実行 entry point を含む backend との役割対応が弱い。
- `app/`: frontend との対比が backend ほど明快でない。
- `internal/` を維持する: アプリケーションでは compiler の import 境界に実益がなく、命名の非対称も残る。
- `pkg/`: 外部 module に公開するライブラリ API と誤読されやすい。

## 影響
- Go import path、ビルド設定、Docker/CI 設定、sqlc 設定、検証 manifest のファイルパスが更新される。
- 現在の構成は `ARCHITECTURE.md` と `CLAUDE.md` に反映する。
- SCL 要素・契約・データ・運用上の振る舞いは不変であり、`spec/scl.yaml` は変更しない。
