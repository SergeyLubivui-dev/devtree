## What this changes

<!-- And why. The what is visible in the diff; the why is not. -->

## Checks

CI runs these on Linux, macOS, and Windows. Running them first is faster than waiting for it:

```bash
gofmt -l .             # must print nothing
go vet ./...
go test -race ./...
go run . check --strict
go run . render        # must leave the working tree clean
go run ./internal/art  # likewise — the documentation pictures are generated
```

- [ ] The checks above pass
- [ ] New behavior has a test, and the test fails without the change
- [ ] User-visible changes are reflected in the README and in `docs/`
- [ ] If it changes what a user does or sees, the translations in `docs/i18n/` are updated too
