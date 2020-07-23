package config

// Max amount of lines you wish commands to output (before hopefully, going into an unimplemented more buffer)
var MaxLines = 5

// Database generated from gendb
var Songdb = "./media.db"

// Playback Volume level
var VolumeLevel float32

var CmdPrefix = "!" // TODO: Set from configuration file
