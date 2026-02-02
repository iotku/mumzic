# mumzic
Music Bot for mumble servers, can play youtube/soundcloud or local files.
WIP

## Getting Started

### Dependencies

Base Reqirements: go / ffmpeg / yt-dlp / sqlite3 

#### 2026 Update: 

We now require opus development headers to build with https://github.com/hraban/opus/tree/v2

Development Headers: opus / opusfile

Build tools: pkg-config

Debian, Ubuntu, ...:
```sh
sudo apt-get install pkg-config libopus-dev libopusfile-dev
```

Fedora:
```sh
sudo dnf install opus-devel opusfile-devel pkgconfig
```

Bazzite (Immutable Fedora w/ Homebrew)
```sh
export PKG_CONFIG_PATH=$(brew --prefix)/lib/pkgconfig:$PKG_CONFIG_PATH
brew install pkg-config opus opusfile
```

Mac:
```sh
brew install pkg-config opus opusfile
```

Windows: Consider using Docker or WSL Ubuntu.

### Building 
Until I can figure out modifying modules properly

`$ git clone https://github.com/iotku/mumzic/`

You can `go build` which should pull in my modified gumble which has stereo support

### Running

`mumzic -insecure -server [hostname or ip]`

For additional options (such as setting the **username** or **password**), see `mumzic -help`

Note: Here we used the `-insecure` flag, to (hopefully) avoid the pain that comes with setting up certificates

    - Currently we don't check for credentials in the `.env` file when running directly

### Docker Compose

In the root directory:

Copy `.env-example` -> `.env` with server information and credentials.

``` docker compose up ```

Use `--build` to rebuild the image.

See docker-compose.yml if you want to enable bindings for a local media.db

### Usage / Commands
See [usage.md](https://github.com/iotku/mumzic/blob/master/USAGE.md)

## Stereo Audio
Mumble Client 1.4.x or higher required, Enable Positional Audio and Headphones checkbox in Mumble Client Audio Output settings.

For stereo audio to work see the building instructions above which should *hopefully* pull in my modified gumble with stereo output.
Using `go get` / `go install` WONT work becasue it does not respect the replace method in go.mod.
