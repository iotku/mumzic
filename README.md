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
