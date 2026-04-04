# Hybrid Migrations

Reference: [GH: VojtechVitek - Migration order problem explained](https://github.com/pressly/goose/issues/63#issuecomment-428681694)

## Notes from goose

By default, if you attempt to apply missing (out-of-order) migrations `goose` will raise an error. However, If you want to apply these missing migrations pass `goose` the `-allow-missing` flag, or if using as a library supply the functional option `goose.WithAllowMissing()` to `Up`, `UpTo` or `UpByOne`.

However, we strongly recommend adopting a hybrid versioning approach, using both timestamps and sequential numbers. Migrations created during the development process are timestamped and sequential versions are ran on production. We believe this method will prevent the problem of conflicting versions when writing software in a team environment.

To help you adopt this approach, `create` will use the current timestamp as the migration version. When you're ready to deploy your migrations in a production environment, we also provide a helpful `fix` command to convert your migrations into sequential order, while preserving the timestamp ordering. We recommend running `fix` in the CI pipeline, and only when the migrations are ready for production.
