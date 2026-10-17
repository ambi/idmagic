# Sourcing のデータ

この文書は、Sourcing の Aggregate を扱う。

## 集約

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `ScimUserRef` | テナント、SCIM の ID、内部の User の ID | `User` を参照する |
| `ScimGroupRef` | テナント、SCIM の ID、内部の Group の ID | `Group` を参照する |

- **判断**：取り込み元との対応（`ScimUserRef`、`ScimGroupRef`）は `User` と `Group` の Aggregate に埋め込まず、このモジュールの独立した Aggregate とする。埋め込むと、IdManagement の主体のモデルが取り込み元の存在を知ることになり、取り込み元の変更が IdManagement へ波及するからである。

用語集の `IdentitySource`、`IngestionRun`、`SourceCursor` は、取り込み元の種類が増えたときの概念であり、現在の `scim` の機能は実体を持たない。
モジュールのルートにはファサードと組み立てだけを置き、複数の取り込み元に実在する共通点が判明するまでは、共通の機構を作らない。
