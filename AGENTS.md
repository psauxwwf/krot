# AGENTS.md

Instructions for AI coding agents working with this codebase.

## Naming

- Do not use the `Cfg` abbreviation in names.
- Always use the full `Config` form instead.
- Do not use the `Urls` form in Go identifiers.
- Always use the `URLs` form for that acronym instead.

<!-- opensrc:start -->

## Source Code Reference

Source code for dependencies is available in `opensrc/` for deeper understanding of implementation details.

See `opensrc/sources.json` for the list of available packages and their versions.

Use this source code when you need to understand how a package works internally, not just its types/interface.

### Fetching Additional Source Code

To fetch source code for a package or repository you need to understand, run:

```bash
opensrc <package>           # npm package (e.g., npx opensrc zod)
opensrc pypi:<package>      # Python package (e.g., npx opensrc pypi:requests)
opensrc crates:<package>    # Rust crate (e.g., npx opensrc crates:serde)
opensrc <owner>/<repo>      # GitHub repo (e.g., npx opensrc vercel/ai)
```

<!-- opensrc:end -->
