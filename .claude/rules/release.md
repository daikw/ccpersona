# Release Procedure

## Prerequisites

- All CI checks passed on main branch
- PR merged to main

## Steps

1. **Check latest tag**
   ```bash
   git tag --sort=-v:refname | head -5
   ```

2. **Create annotated tag**
   ```bash
   git tag -a v0.1.X -m "Release description"
   git push origin v0.1.X
   ```

3. **Create GitHub Release**
   ```bash
   gh release create v0.1.X --title "v0.1.X" --notes "$(cat <<'EOF'
   ## What's New

   ### Feature Name

   Description of the feature.

   ### Other Changes

   - Change 1
   - Change 2

   **Full Changelog**: https://github.com/daikw/ccpersona/compare/v0.1.Y...v0.1.X
   EOF
   )"
   ```

4. **Verify goreleaser**
   ```bash
   gh run list --limit 3
   ```
   - goreleaser automatically builds binaries for darwin/linux/windows
   - Binaries are uploaded to the release page

## Version Numbering

- `v0.1.X` - Current development phase
- Increment X for each release
- Use semantic versioning when reaching v1.0.0

## Release Notes Format

- Write in Japanese (project language)
- Include:
  - New features with examples
  - Breaking changes (if any)
  - Bug fixes
  - Link to full changelog
