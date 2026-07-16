# Retrieving Work Item Context

While the Work Item YAML files are the canonical records, AI agents should initially read only the following fields:

1. `motivation`
2. `scope`
3. `out_of_scope`
4. `change_kind`
5. `initial_context`
6. `affected_spec` or `spec_impact`
7. `verification`
8. `risk`

Access large `completion` fields or validation evidence only when historical audits or past verification results are required.
For `initial_context`, prioritize directory-based `features` and feature directories rather than long lists of individual files. List specific file paths only when they represent exceptional entry points located outside the feature directories.
Resolve every `affected_spec` entry as a context-qualified SCL element reference before reading implementation. Treat `affected_guarantees` as completed-record history only.
