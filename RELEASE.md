# Releasing `github.com/plaidly/plaidly-go`

Go modules are not uploaded anywhere — a version is "published" the moment a
semver tag exists in the repo and the module proxy fetches it. There are **no
secrets** to configure.

Two workflows back this:

- `.github/workflows/ci.yml` — on every push to `main` and PR: `go mod tidy`
  check, `go vet ./...`, `go test ./...`.
- `.github/workflows/release.yml` — on a `v*` tag: verifies the tag is valid
  semver, runs vet + tests, then warms `proxy.golang.org` so `go get` resolves
  the new version right away.

## Cutting a release

1. Ensure `main` is green and `go.mod` declares the right module path
   (`github.com/plaidly/plaidly-go`).
2. Tag with a semantic version and push:
   ```bash
   git tag v0.2.1
   git push origin main --tags
   ```
3. The **Release** workflow gates the tag on tests and pings the proxy.

## Verify it resolves

```bash
GOPROXY=proxy.golang.org go install github.com/plaidly/plaidly-go@v0.2.1   # or:
go get github.com/plaidly/plaidly-go@v0.2.1
```

Notes:
- Tags are effectively immutable once the proxy caches them — never move or
  delete a published tag; bump the patch instead.
- `v2`+ major versions require a `/v2` suffix on the module path in `go.mod`
  and import paths (Go's semantic import versioning). Not needed for `v0`/`v1`.
