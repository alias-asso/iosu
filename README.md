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

TODO

## Development

TODO
