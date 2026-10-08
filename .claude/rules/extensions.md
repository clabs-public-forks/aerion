---
paths:
  - "extensions/**"
  - "internal/core/**"
  - "internal/extensions/**"
  - "internal/kit/**"
  - "app/coreimpl*.go"
  - "app/extension_*.go"
  - "frontend/src/lib/components/kit/**"
---

# Extension changes

Before committing a change to an extension, the core API it consumes, or the backend or UI kit, check it against `docs/EXT_RULES.md`. If the change would break a rule, rethink the design instead.

`docs/EXTENSIONS.md` is the full reference (about 170 KB). Read its Contents section, then only the sections you need.
