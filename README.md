# mumzic
Music Bot for mumble servers, can play youtube/soundcloud or local files.
WIP

## Getting Started

### Building
Base Reqirements: go / ffmpeg / yt-dlp / sqlite3

`$ go get github.com/iotku/mumzic/`

You can then either `go build` or `go install`

### Running

`mumzic -insecure -server [hostname or ip]`

For additional options (such as setting the **username** or **password**), see `mumzic -help`

Note: Here we used the `-insecure` flag, to (hopefully) avoid the pain that comes with setting up certificates

### Usage / Commands
See [usage.md](https://github.com/iotku/mumzic/blob/master/USAGE.md)

## Known Mumble Limitations (Not my fault, for once.)
* No stereo audio (See https://github.com/mumble-voip/mumble/issues/2829 & https://github.com/layeh/gumble/issues/51)
