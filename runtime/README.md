# Runtime (pinned)

Submodules pinned at release time:

- `d-research-skill/` — research scripts, templates, schemas (`570d30b`)
- `aleph/` — simulation engine, schema, templates, no full KB (`754937c`)

Initialize after clone:

```powershell
git submodule update --init --recursive
```

v0.1 expects Python 3.10+ and Node 20+ on the host machine.

Runtime resolution does not depend on CWD. The CLI resolves `runtime/` relative to the executable, an ancestor directory, or `D_RESEARCH_RUNTIME`.