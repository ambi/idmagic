---
status: completed
---

# Inter、tw-animate-css、clsx の直接依存を外す

規範仕様の差分はない（`spec-diff`：no normative specification change against main）。
フロントエンドの直接依存から `@fontsource-variable/inter`、`tw-animate-css`、`clsx` を外した。
英数字は `Noto Sans JP` 以降の OS のフォントで表示し、select と dropdown menu の popup は開閉の動きを持たず即座に表示と消去を行う。
`cn()` は `twMerge(...inputs)` だけで合成し、出力は変わらない。
Base UI の `className` は状態を受け取る関数も許すが、clsx も `twMerge` も関数を空文字にするため、`cn()` で合成する select と dropdown menu の部品は `StringClassNameProps` で `className` を文字列に限った。
E2E には、dropdown menu を閉じた後に popup が DOM に残らない表明を足した。
`verify-changed` の初回は既知の不安定な単体テスト（wi-63323）で失敗し、最終差分のまま `verify-ui` を再実行して通った。
