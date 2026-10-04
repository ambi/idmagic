# 起動時設定の設計

この文書は、[起動時設定](README.md)の要件を保証する仕組みを扱う。

## アーキテクチャ

`FeatureDefinition` は、`FeatureID`、`FeatureVersion`、`FeatureMaturity`、`DefaultEnablement`、依存する `FeatureID`、`UpdatePolicy`、任意の `SpecificationReference` を持つ不変の値である。
起動処理が静的な `FeatureRegistry` を渡し、設定ファイルやデータベースから機能の定義を増やさない。
レジストリの検証は、識別子と版のない名前の重複、存在しない依存、循環、不正なデフォルトの有効化を、すべて集約して返す。

`ResolveFeatures(registry, explicitEnable, explicitDisable)` は、時刻、乱数、永続化に依存しない決定的な計算である。
明示した指定をデフォルトの値へ重ね、有効な機能の依存の閉包を求める。
存在しない識別子、同じ機能の有効化と無効化の併記、明示的に無効化した依存を必要とする選択は、すべての設定の検証と同じく、作用のある初期化の前に集約して拒否する。

解決の結果は、有効な `FeatureDefinition`、起動の警告、版付きの運用のメタデータを一度に返す。
設定の参照文書と `/health` はこのレジストリと解決の結果から導き、手書きの機能の一覧は設けない。
警告と運用のメタデータは、識別子、版、成熟度、更新の方針だけを含み、環境変数の生の値を含まない。
