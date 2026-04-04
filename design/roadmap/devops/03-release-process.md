# Release Process

This doc covers tagging releases, writing release notes that include migration information, and the customer's upgrade experience.


## Tagging a release

After `goose fix` on the release branch and all tests pass:

```bash
git tag -a v1.2.0 -m "Release v1.2.0"
git push origin v1.2.0
```

The tag must point to a commit that contains:
- Sequential migrations (post-`goose fix`)
- Passing `make test-integration`
- Release notes documenting migration changes


## Release notes template

Every release that includes migrations should document them explicitly. Create release notes (GitHub release, CHANGELOG.md, or both) following this structure:

```markdown
## v1.2.0

### What's new
- Brief description of features and fixes

### Database migrations

This release adds 2 new migrations (00003, 00004).

| Migration | Description | Backward-compatible? |
|-----------|-------------|----------------------|
| 00003_add_tokens.sql | Creates the `tokens` table | Yes |
| 00004_add_refresh_tokens.sql | Creates the `refresh_tokens` table | Yes |

**Upgrade:**

    goose -dir sql/schema up

**Rollback (if needed):**

    goose -dir sql/schema down-to 00002

### Upgrade instructions

1. Apply migrations to staging: `goose -dir sql/schema up`
2. Smoke test staging
3. Apply migrations to production: `goose -dir sql/schema up`
4. Deploy the new application version

### Breaking changes

None. All migrations are additive. The previous application version (v1.1.0)
will continue to work against the new schema.
```

When a release has no migrations, say so explicitly:

```markdown
### Database migrations

No schema changes in this release.
```


## Backward compatibility

Prefer additive migrations. These are backward-compatible — the previous application version continues to work against the new schema:

- Add a table
- Add a column with a default
- Add an index

These are NOT backward-compatible — the previous application version will break:

- Drop a column
- Rename a column
- Change a column type

When a destructive change is unavoidable, split it across two releases:

1. **v1.2.0:** Add the new column. Update the app to write to both old and new columns.
2. **v1.3.0:** Drop the old column. Update the app to use only the new column.

This gives the customer a safe upgrade path and a rollback path at each step. Document the two-release plan in both sets of release notes so the customer understands the sequence.


## The customer's upgrade workflow

A cautious customer upgrading from v1.1.0 to v1.2.0:

1. Read the release notes — check for migration changes and breaking changes
2. `git checkout v1.2.0`
3. `goose -dir sql/schema up` against staging
4. Deploy the new binary to staging, smoke test
5. `goose -dir sql/schema up` against production
6. Deploy the new binary to production

If staging fails:
- `goose -dir sql/schema down-to <previous>` to roll back the migration
- Report the issue — production is untouched

If production fails after migration:
- If migrations are backward-compatible: roll back the app to v1.1.0, investigate
- If not: `goose down-to <previous>` and roll back the app

Skipping versions (v1.0.0 → v1.2.0) works — goose applies all unapplied migrations in order. But the customer should read release notes for all skipped versions to understand cumulative changes.


## Versioning convention

Follow [semantic versioning](https://semver.org/):

- **Patch** (v1.2.1): Bug fixes, no migrations
- **Minor** (v1.2.0): New features, additive migrations (backward-compatible)
- **Major** (v2.0.0): Breaking changes, destructive migrations, or API changes

A release that includes a non-backward-compatible migration is at minimum a minor version bump, and should be a major bump if the customer can't roll back without data loss.


## Reference

- [Semantic Versioning](https://semver.org/)
- [devops/02-migration-discipline.md](02-migration-discipline.md) — branch workflow and `goose fix` timing
