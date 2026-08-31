# Installation

Pick whichever fits how you already manage software on the machine that'll
run this. All six install the same binary; none of them need a Go
toolchain except the first.

## go install

```shell
go install github.com/alrayyes/linkwarden-obsidian-sync/cmd/linkwarden-obsidian-sync@latest
```

Builds and installs straight from the module proxy, no separate download.
Pin a version instead of `@latest` for a reproducible install:
`go install .../cmd/linkwarden-obsidian-sync@v2.3.1`.

## Prebuilt binary

Grab the `linux`/`darwin`, `amd64`/`arm64` archive that matches your
machine from the [latest
release](https://github.com/alrayyes/linkwarden-obsidian-sync/releases/latest),
extract it, and put the binary on your `PATH`:

```shell
tar -xzf linkwarden-obsidian-sync_linux_amd64.tar.gz
sudo install -m755 linkwarden-obsidian-sync /usr/local/bin/
```

## Debian/Ubuntu (.deb)

Grab the `.deb` matching your architecture from the [latest
release](https://github.com/alrayyes/linkwarden-obsidian-sync/releases/latest),
then:

```shell
sudo dpkg -i linkwarden-obsidian-sync_*_linux_amd64.deb
```

Installs the binary and both man pages (`man linkwarden-obsidian-sync`,
`man linkwarden-obsidian-sync-init`).

## Fedora/RHEL (.rpm)

Same release page, the `.rpm` instead:

```shell
sudo rpm -i linkwarden-obsidian-sync_*_linux_amd64.rpm
```

Same contents as the `.deb` above.

## Arch (AUR)

```shell
yay -S linkwarden-obsidian-sync
```

Or with any other AUR helper, or `git clone
https://aur.archlinux.org/linkwarden-obsidian-sync.git && cd
linkwarden-obsidian-sync && makepkg -si` by hand. Builds from source
against the tagged release; there's no `-git` variant. The
[PKGBUILD](https://aur.archlinux.org/packages/linkwarden-obsidian-sync)
tracks each GitHub release automatically — `packaging/aur/` in this repo
is the template it's generated from.

## Nix / NixOS

```shell
nix run github:alrayyes/linkwarden-obsidian-sync
```

Or add it as a flake input, or `nix profile install
github:alrayyes/linkwarden-obsidian-sync` to install it into your profile.
Builds straight from this repo's own `flake.nix` with `buildGoModule` —
no nixpkgs submission, so nothing to wait on there.

## Docker

`docker run ghcr.io/alrayyes/linkwarden-obsidian-sync:latest`, configured
entirely through environment variables — see
[docs/USAGE.md](USAGE.md#docker) for the full command, what to mount, and
the configuration reference.

## After installing

Every path above ends with the same binary. From here:

```shell
linkwarden-obsidian-sync init   # writes a config template and tells you where
# edit it: set linkwarden_url and linkwarden_token
linkwarden-obsidian-sync
```

See [docs/USAGE.md](USAGE.md) for what gets added, updated, and removed on
a run, the full configuration reference, and the state file's shape.
