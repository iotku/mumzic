## Command List
Currently the bot defaults to "!" for the command prefix, therefore to use **play** you would type **!play** into the messagebox.
Note: PMing the bot currently doesn't have the prefix for commands, so **!summon** would just be **summon*

### Control
| Command | Info                      | Notes                                         |
|-------------------------------------------------------------------------------------|
| !summon | Bot joins sender's channel| PM the bot if you are not in the same channel | 

### Playback
| Command           | Info                                  | Notes                                                                     |
|-------------------|---------------------------------------|---------------------------------------------------------------------------|
| play [id or URL] | Play track via ID                     | Numberic IDs (found with !search) or Youtube/Soundcloud URL               |
| rand [#]         | Add Random Tracks                     | Random track(s) from filesystem                                           |
| stop             | Stop playing track                    | If you use !play with no arguments; will restart track from beginning     |
| skip [#]         | skip # amount of tracks               | Default 1                                                                 |


### Playlist
| Command           | Info                                  | Notes                                                                     |
|-------------------|---------------------------------------|---------------------------------------------------------------------------|
| list             | Show current track list               |                                                                           |
| search           | Find tracks from local files          |                                                                           |

### Audio
| Command           | Info                                  | Notes                                                                     |
|-------------------|---------------------------------------|---------------------------------------------------------------------------|
| volume (1-9)     | Set Volume                            | Eventually will be percentage based                                       |
| target           | Send audio to you directly            | Works no matter what channel you are in as long as Whispers are enabled   |
| untarget         | Don't send audio to you directly      | Remove you from audio targetting list                                     |

## Generating a local media.db (for local file playback)

Currently "genMusicSQLiteDB" ([found here](https://github.com/iotku/genMusicSQLiteDB)) is used to create a local database of local files for the bot to play.
Without a media.db present only ytdl links will work.

### Create media.db for mumzic
`$ gendb [path/to/music/directory]`
