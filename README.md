# mumzic
Music Bot for mumble servers, can play youtube/soundcloud or local files.
WIP

## Getting Started

### Building
Base Reqirements: go / ffmpeg / yt-dlp / sqlite3

Until I can figure out modifying modules properly

`$ git clone github.com/iotku/mumzic/`

You can `go build` which should pull in my modified gumble which has stereo support

### Running

`mumzic -insecure -server [hostname or ip]`

For additional options (such as setting the **username** or **password**), see `mumzic -help`

Note: Here we used the `-insecure` flag, to (hopefully) avoid the pain that comes with setting up certificates

### Usage / Commands
See [usage.md](https://github.com/iotku/mumzic/blob/master/USAGE.md)

## Stereo Audio
Mumble Client 1.4.x or higher required, Enable Positional Audio and Headphones checkbox in Mumble Client Audio Output settings.

For stereo audio to work see the building instructions above which should *hopefully* pull in my modified gumble with stereo output.
Using `go get` / `go install` WONT work becasue it does not respect the replace method in go.mod.
