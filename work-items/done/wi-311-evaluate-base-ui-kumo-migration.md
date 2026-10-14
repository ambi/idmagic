---
status: completed
archived_from: 7dac9e695840850ceab09c0b13a19056b223cc97
---

# Radix UI + shadcn/ui から Base UI (+ Kumo) への移行要否を調査し、方針を決める

Radix UI + shadcn/ui から Base UI (+ Kumo) への移行要否を実機検証した。
`@cloudflare/kumo` のソースを取得した結果、デザイントークンは非公開の内部生成
スクリプトによる Cloudflare ブランド色ハードコードで、外部からの再定義を前提とした
設計ではないため当初は Kumo を却下し「Base UI を自前でラップし直す」方針を検討した。
`@base-ui/react` を実際に組み込み `button.tsx` / `dropdown-menu.tsx` / `label.tsx` /
`select.tsx` を試作した結果、DropdownMenu の Base UI `Menu` への移行は機械的に可能だが、
(1) Base UI には独立した Label プリミティブが無く `Field.Label` は `Field.Root` 祖先
必須、(2) `Button` の `asChild → render` 変換は非 `<button>` ターゲットに
`nativeButton={false}` の追加指定が要る、(3) 呼び出し側の実件数は 51 箇所・約 34
ファイルで構造変更が必要、という当初想定より詳細な差分があることも判明した。
レビューで「Kumo のブランド色ハードコードは GitHub issue で問題視されていないか」との
指摘を受けて追加調査したところ、Kumo は Shopify Polaris 等と同種の「自社製品用
内製デザインシステムの OSS 公開」であり外部ブランド利用は元々想定外と判明。同時に
ユーザーから「デザイン刷新は歓迎、互換性は不要だが Cloudflare のブランド色を
そのまま被るのは避けたい」との意向を確認し、**最終方針を shadcn/ui の Base UI 版
（`shadcn init -b base` によるソース生成方式）の採用に転換した**。これにより
Kumo のブランド上書き問題も、Base UI 自前ラップの保守負担も回避しつつ、
Dialog/Toast/Combobox 等の不足コンポーネントも同じ枠組みで拡充できる。
この決定に基づく実装は [[wi-312-migrate-radix-to-base-ui]] に分割した。
プロトタイプ変更は検証後に `git checkout` で元の Radix 実装へ revert 済みで、
本 WI 自体はコード変更を残さない。

全文は `git show 7dac9e695840850ceab09c0b13a19056b223cc97:<このファイルのパス>` で読む。
