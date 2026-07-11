# cine

A fast, modular, terminal-native media CLI. `cine` looks up metadata from
Cinemeta (keyless), resolves playable streams through pluggable providers, and
plays them in `mpv` - designed so no single backend can take the whole app down.

## Features

- Keyless metadata via Cinemeta (already IMDb-keyed)
- Pluggable stream providers with parallel query + fallback
- Quality-aware ranking (resolution, HDR/DV, source, codec)
- `mpv` playback, with a peerflix bridge for magnets
- Watch history and favorites (SQLite)
- Download mode that hands magnets to your torrent client
- `cine doctor` to diagnose your setup

## Install

Prebuilt binaries are attached to each GitHub release - download the archive for
your platform, extract it, and put `cine` on your PATH.

From source (Go 1.22+):

    git clone https://github.com/yourname/cine
    cd cine
    make build
    ./cine version

## Requirements

- Required: `mpv` (player) and `curl` (fetching)
- Optional: `peerflix` (magnet streaming) and `chafa` (poster preview)

      npm install -g peerflix

## Configure

Copy `config.example.yaml` to `~/.config/cine/config.yaml` and enable the
providers you want. The bundled `demo` provider streams Creative-Commons movies
and needs no setup, so you can verify playback immediately.

## Usage

    cine search "Breaking Bad"
    cine play "The Matrix"
    cine play "Severance S01E01"
    cine play "Dune" --download
    cine history
    cine favorites add "Interstellar"
    cine doctor

## Providers

`cine` ships as a plugin framework. The core repo hardcodes no unofficial
sources - you configure your own provider endpoints. Keep providers that touch
unofficial indexers out of the public core, or gate them behind user-supplied
config.

## Troubleshooting

Run `cine doctor` first. Common issues:

- Playback fails on WSL/headless: set `player_args: ["--vo=x11"]` or `--vo=tct`.
- Magnets never start: install `peerflix` and set `resolver.type: peerflix`.
- HLS returns 403: use a provider that forwards Referer headers (consumet/direct).

## Development

    make fmt     # format
    make vet     # go vet
    make test    # unit tests with -race
    make lint    # golangci-lint
    make build   # reproducible static binary

## License

MIT - see LICENSE.
