# iosu

The programming contest platform of the [ALIAS](https://alias-asso.fr) student
association. Each contestant gets their own input for a problem and submits the
answer they compute from it.

Problem statements are markdown files on disk; everything else lives in a SQLite
database.

## Building

On NixOS, or anywhere with Nix:

```sh
nix build     # both binaries in ./result/bin; also runs the tests
```

Everywhere else — Linux and OpenBSD — there is a Makefile that works with both
GNU make and OpenBSD's make:

```sh
make          # builds iosud and iosu
make check    # gofmt, go vet, go test
make install  # into $DESTDIR$PREFIX, default /usr/local
make help     # every target
```

The only build dependency is Go. The SQLite driver is pure Go, so nothing needs
a C compiler and cross-compiling is just a matter of setting the target:

```sh
make GOOS=openbsd GOARCH=amd64
```

The driver supports Linux and OpenBSD on amd64 and arm64, plus linux/arm,
FreeBSD, netbsd/amd64, macOS and Windows.

Two notes for OpenBSD: the binaries link against libc there, as the kernel
requires, so they are not static the way the Linux ones are; and `make
test-race` will not work, because Go's race detector needs cgo and has no
OpenBSD support. Plain `make test` is fine.

## Configuring

Copy `config.example.toml` to `/etc/iosu/config.toml` and fill it in. Every
value can also come from the environment, which takes precedence: `IOSU_PORT`,
`IOSU_JWT_KEY`, `IOSU_ADMIN_PASSWORD`, `IOSU_DATA_DIR`, `IOSU_DB_PATH`.

`jwt_key` signs session cookies and must be at least 32 characters:

```sh
head -c 32 /dev/urandom | base64
```

Changing it logs everyone out. **Never commit the real config file.**

`iosud` terminates plain HTTP and expects to sit behind a TLS-terminating
reverse proxy. Session cookies are marked `Secure` unless `dev_mode` is set.

## Running

```sh
iosud -c /etc/iosu/config.toml
```

The first start creates the schema, an `admin` account using
`default_admin_password`, and placeholder site content. Later starts leave all
three alone, so changing the admin password sticks:

```sh
iosu user passwd -username admin
```

## Data directory

```
<data_directory>/<contest-slug>/<problem-slug>/part1.md
                                              /part2.md
                                              /img/figure.png
```

Part *N+1* is only shown once a contestant has solved part *N*. Slugs are
restricted to lowercase letters, digits and dashes, since they are used as path
segments.

## Administration

Run `iosu` for the full list. The usual sequence for a new contest:

```sh
iosu difficulty create -name facile -points 10
iosu contest create -slug 2026 -name "IO/SU 2026" \
    -start-time "2026-03-25 18:00:00" -end-time "2026-03-27 20:00:00"
iosu problem create -contest 2026 -slug melange-bits -name "On a mélangé les bits" \
    -difficulty moyen -author florianclume -parts 1
iosu user batch-create -i participants.csv     # CSV with username,email columns
iosu user pending -url https://iosu.example.org # activation links to send out
iosu config update -site-title "IO/SU" -current-contest 2026
iosu config import -help help.md -rules rules.md
```

Per-contestant inputs and answers are produced outside the platform and imported
from a directory tree:

```
<directory>/<problem-slug>/<username>/input.txt
                                     /output1.txt, output2.txt, ...
```

```sh
iosu contest data -contest 2026 -directory ./generated
```

Re-running the import overwrites what is already stored.

## Development

```sh
nix develop   # go, sqlc, sqlite, gopls, staticcheck, govulncheck
```

```sh
go test -race ./...
go vet ./... && staticcheck ./...
```

Without Nix, `make check` covers gofmt, vet and the tests.

Queries live in `internal/store/query/*.sql` and the schema in
`internal/store/migrations/*.sql`. Both are inputs to
[sqlc](https://sqlc.dev), which generates `internal/store/sqlc` — never edit
that directory by hand:

```sh
sqlc generate
```

In the dev shell sqlc's version is pinned by `flake.lock`. Outside it, `make
generate` fetches the same version, pinned as `SQLC_VERSION` in the `Makefile` —
keep the two in step, or the generated code will differ. CI checks that the
committed output matches either way.

To change the schema, add a new `NNN_description.sql` migration rather than
editing an existing one. `iosud` applies pending migrations at startup and
tracks the version in SQLite's `PRAGMA user_version`.

## Layout

| | |
|---|---|
| `cmd/iosud`, `cmd/iosu` | entry points, kept thin |
| `internal/config` | TOML config and environment overrides |
| `internal/store` | schema, migrations and generated queries |
| `internal/app` | domain logic, shared by the server and the CLI |
| `internal/web` | routing, session handling, HTML rendering; `static/` holds the CSS and JS served as-is |
| `internal/cli` | the `iosu` subcommands |
| `design/` | standalone HTML mock-ups, not served |
