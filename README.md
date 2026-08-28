# iosu

IO/SU is a programming contest organised each year by [ALIAS](https://alias-asso.fr). This is the code for the contedt's platform.

## Building

A Makefile is present to help with all the build tasks.

```sh
make          
make check 
make install
make help
```

Since ALIAS infrastructure is on an OpenBSD server you can build the binaries for this target using the following command:

```sh
make GOOS=openbsd GOARCH=amd64
```

## Configuration

Copy `config.example.toml` to `/etc/iosu/config.toml` and fill it in. Every
value can also come from the environment, which takes precedence.

## Running

Run the daemon:

```sh
iosud
# or with a custom config file path
iosud -c /path/to/config.toml
```

The first start creates the schema, an `admin` account using
`default_admin_password`, and placeholder site content. Please immediately change the admin password using:

```sh
iosu user passwd -username admin
```

## Administration

Run `iosu` for the full list. The usual sequence for a new contest:

```sh
iosu difficulty create -name facile -points 10
iosu contest create -slug 2026 -name "IO/SU 2026" \
    -start-time "2026-03-25 18:00:00" -end-time "2026-03-27 20:00:00"
iosu problem create -contest 2026 -slug your-slug -name "Full Name" \
    -difficulty moyen -author problem_author -parts 1
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

TODO
