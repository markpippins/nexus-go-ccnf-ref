# Canonical Serialization Contract — CCNF

This document defines the immutable serialization rules for CCNF.
All golden vectors, hashes, signatures, and replay guarantees depend on
bitwise adherence to this contract.

**Changing this file is a breaking protocol change.**
Any change requires a CCNF version increment.

---

## JSON Object Keys

- Ordered lexicographically
- Bytewise UTF-8 comparison
- Applied recursively at all nesting levels

---

## Whitespace

- Compact JSON only
- No spaces
- No indentation
- No trailing newline

---

## Numbers

- Integers → JSON number without `.0`
- Floats → IEEE-754 double precision
- Fixed notation only
- Locale independent `.`
- Scientific notation forbidden

---

## Strings

- UTF-8 encoded
- NFC normalized at ingress
- BOM stripped
- Zero-width characters removed
- Case preserved

---

## Null vs Absent

All fields MUST be present:

- nullable → `null`
- arrays → `[]`
- objects → `{}`

Missing fields are invalid.

---

## Timestamps

- Canonical form: epoch seconds (`int64`)
- ISO-8601 inputs converted during normalization
- No timezone representation allowed

---

## Arrays

- `ordered:true` → preserve input order
- otherwise → lexicographically sorted

---

## Hash

```
SHA256(canonical_UTF8_bytes)
```

No trailing newline included.
