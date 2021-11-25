## Command List
| Command           | Info                                  | Notes                                                                     |
|-------------------|---------------------------------------|---------------------------------------------------------------------------|
| !play [id or URL] | Play track via ID                     | Numberic IDs (found with !search) or Youtube/Soundcloud URL               |
| !search           | Find tracks from local files          |                                                                           |
| !list             | Show current track list               |                                                                           |
| !skip [#]         | skip # amount of tracks               | Default 1                                                                 |
| !stop             | Stop playing track                    | If you use !play with no arguments; will restart track from beginning     |
| !rand [#]         | Add Random Tracks                     | Random track(s) from filesystem (Limit 5)                                 |
| !volume (1-9)     | Set Volume                            | Eventually will be percentage based                                       |

## Generating a local media.db (for local file playback)

Currently "gendb" ([found here](https://github.com/iotku/genMusicSQLiteDB)) is used to create a local database of Music files for the bot to play and is built seperately (go build/go install in the gendb directory).

### Create media.db for mumzic
`$ gendb [path/to/music/directory]`

### Supported Formats
Currently the gendb program only looks for .flac files because I'm a snob, this should be able to change in the future.
