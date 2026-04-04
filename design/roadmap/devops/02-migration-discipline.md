# Migration Discipline for Teams

When multiple developers are creating database migrations concurrently, you need rules to prevent conflicts, broken environments, and silent schema drift. This doc covers the practical problems and the conventions that solve them.


## The conflict problem

Developer A creates `000003_add_chirps.sql`. Developer B creates `000003_add_tokens.sql`. Both work fine on their own branches. Both merge to main. Now you have two migrations with the same version number, and goose doesn't know which to run first — or it picks one arbitrarily and the other fails because it expected a different schema state.

This is the fundamental problem with sequential numbering in a team.


## Timestamps solve ordering

Goose supports timestamp-based migration filenames:

```
20260404120000_add_chirps.sql
20260404120100_add_tokens.sql
```

Two developers creating migrations at different times will never collide on the version number. Goose applies them in timestamp order, which matches the order they were written.

Use `goose create` to generate timestamped filenames:

```bash
goose -dir sql/schema create add_chirps sql
# creates sql/schema/20260404120000_add_chirps.sql
```

This is your default during development. Every new migration gets a timestamp.


## The golden rule: never edit applied migrations

Once a migration has been applied to a shared environment (staging, production, or even a teammate's local database), treat it as immutable. If you need to change something:

- Create a new migration that alters the table
- Don't modify the existing migration file

Why: goose tracks which migrations have been applied by version number. If you change the contents of an already-applied migration, goose won't re-run it — it thinks it's already done. The schema on existing databases will silently diverge from what the migration file says.

The only exception: if a migration has never left your local branch, you can edit or delete it freely.


## Out-of-order migrations

When Developer A's branch merges first with migration `20260404120000`, and Developer B's branch merges later with migration `20260403110000` (an earlier timestamp), goose sees an "out of order" migration — a version that's older than the latest applied one.

By default, goose rejects this. You have two options:

**Option 1: `--allow-missing` flag.** Tells goose to apply any unapplied migrations regardless of order:

```bash
goose -dir sql/schema --allow-missing up
```

This is safe when the migrations are independent (they touch different tables). It's dangerous when they depend on each other's schema changes.

**Option 2: Rebase and re-timestamp.** Before merging, rebase your branch and recreate the migration with a current timestamp. This keeps the ordering clean but requires manual intervention.

For a small team, `--allow-missing` is usually fine. For a larger team, establish a convention: always rebase your migration timestamps before merging to main.


## Migration lifecycle: branches through production

The hybrid strategy only makes sense when you understand where each operation happens relative to your git workflow and deployment pipeline.

```
feature branch → PR / CI → merge to main → staging → release branch → production
```

### Feature branch

Create migrations with `goose create` — always timestamped. Run `make test-integration` locally. The testcontainer applies all migrations from scratch against a fresh database. Commit the timestamped migration files.

Do NOT run `goose fix` here. It renames every migration file in the repo, which guarantees merge conflicts with every other branch that touches migrations.

### Pull request / CI

CI runs `make test-integration`. The testcontainer applies all migrations from scratch against a fresh database. If the new migration conflicts with the existing schema, it fails here.

During review, ask: is this migration independent of other in-flight branches? If two branches both add columns to the same table, they need to be coordinated — that's a people problem, not a tooling problem.

Note: CI always runs against a fresh database, so migration ordering doesn't matter at this stage. The out-of-order problem only surfaces when applying migrations incrementally against a database that already has some applied.

### Merge to main

Timestamped migrations accumulate on main. Their order may be non-sequential across branches — Developer B's earlier timestamp merges after Developer A's later one. This is expected and harmless as long as the migrations are independent.

### Staging deployment

Staging is the first environment with a persistent database — migrations are applied incrementally, not from scratch. Run:

```bash
goose -dir sql/schema --allow-missing up
```

`--allow-missing` because main may have accumulated out-of-order timestamps from concurrent merges. Staging is where you verify that migrations work incrementally, not just from scratch. If staging breaks, you fix it before it reaches production.

### Release branch

When you're ready to cut a release, create a release branch from main:

```bash
git checkout -b release/v1.2 main
goose -dir sql/schema fix
make test-integration    # verify nothing broke
git add sql/schema/
git commit -m "fix: renumber migrations for release v1.2"
```

Why on a release branch, not on main:
- `goose fix` renames files. Doing it on main creates merge conflicts with every open feature branch that has migrations.
- The release branch is frozen — no new migrations are being added. Safe to renumber.
- Production gets clean sequential numbering. Easy to audit, easy to reason about.

### Production deployment

Production runs:

```bash
goose -dir sql/schema up
```

No `--allow-missing` needed — the release branch has clean sequential numbering. Migrations apply in order. If something fails, `goose down` rolls back the last migration.

### Post-release

Merge the release branch back to main so main reflects the sequential numbering going forward:

```bash
git checkout main
git merge release/v1.2
git push
```

Feature branches rebase onto the updated main. Any timestamped migrations on those branches now sit after the sequential ones — which is correct.


## Schema snapshot verification (optional)

For extra safety, you can commit a `schema.sql` snapshot and verify it in CI:

```bash
# dump the schema after running all migrations against a fresh database
pg_dump --schema-only --no-owner --no-privileges chirpy_test > schema.snapshot.sql

# diff against the committed snapshot
diff schema.snapshot.sql sql/schema.sql
```

If the diff is non-empty, someone changed a migration without updating the snapshot, or two migrations produced an unexpected combined result.

This is overkill for a solo project or small team, but valuable when:
- Multiple developers are creating migrations weekly
- You want to catch accidental migration edits
- You need a single file that shows the current schema at a glance

### Makefile target

```makefile
sql-snapshot:
	pg_dump --schema-only --no-owner --no-privileges chirpy_test > sql/schema.snapshot.sql

sql-verify-snapshot:
	@echo "Running migrations against fresh database..."
	# (requires a test database — integrate with testcontainers or a CI Postgres)
	@diff sql/schema.snapshot.sql <(pg_dump --schema-only --no-owner --no-privileges chirpy_test) \
		&& echo "Schema snapshot is current" \
		|| (echo "Schema snapshot is stale — run 'make sql-snapshot'" && exit 1)
```


## Summary of conventions

| Rule | Where | Why |
|------|-------|-----|
| Use `goose create` for new migrations | Feature branch | Timestamps prevent version collisions |
| Never edit an applied migration | Everywhere | Goose won't re-run it; schema silently drifts |
| `--allow-missing` for incremental apply | Staging | Handles out-of-order timestamps from concurrent merges |
| `goose fix` on release branch only | Release branch | Clean sequential numbering without disrupting feature branches |
| `goose up` (no flags) for production | Production | Sequential order, clean, auditable |
| Merge release branch back to main | Post-release | Main reflects the renumbered files going forward |
| Commit a schema snapshot | Optional, CI | Catches drift and accidental edits |

### Makefile targets

```makefile
sql-create:
	@read -p "Migration name: " name; \
	goose -dir sql/schema create $$name sql

sql-fix:
	goose -dir sql/schema fix
```


## Reference

- [goose: hybrid versioning](https://pressly.github.io/goose/blog/2022/overview-sql-file-migrations/)
- [goose: allow-missing](https://pressly.github.io/goose/documentation/faq/#what-is-the-allow-missing-flag)
- [GH: Migration order problem explained](https://github.com/pressly/goose/issues/63#issuecomment-428681694)
- [devops/03-release-process.md](03-release-process.md) — tagging, release notes, and the customer's upgrade experience
