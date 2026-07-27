# Development Practices

This document defines repository-wide practices for making and testing changes. It complements the
normative behavior in SCL, the current design in `ARCHITECTURE.md`, and command instructions in the
root and directory `README.md` files.

## Frontend test locale

Frontend tests use English as their default locale. Ordinary behavior tests assert English labels and
messages so they exercise the same fallback language as the product.

Set `locale: 'ja'` only when the test's explicit purpose is to verify Japanese localization. A Japanese
localization test should say so in its name and should remain separate from ordinary behavior tests.
Presentation tests rendered without a locale provider also fall back to English. Tests that need a
router should use `renderWithRouter`, whose default locale is English, and pass `{ locale: 'ja' }` only
for an explicit Japanese-localization case.

Every translation dictionary must define the same keys for Japanese and English. The dictionary
consistency test remains the mechanical guard for missing translations.
