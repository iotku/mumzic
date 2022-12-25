package commands

import (
	"testing"

	"github.com/iotku/mumzic/config"
)

var Config config.Config

func init() {
	Config = *config.NewConfig("") // TODO: cleanup created database on disk
}

func TestGetCommandAndArg(t *testing.T) {
	// message with nickname as prefix
	got1, got2 := getCommandAndArg("MusicButt command argument", "MusicButt", &Config)
	if got1 != "command" || got2 != "argument" {
		t.Errorf("got %q %q  wanted %q %q", got1, got2, "command", "argument")
	}

	got1, got2 = getCommandAndArg("MusicButt command", "MusicButt", &Config)
	if got1 != "command" || got2 != "" {
		t.Errorf("got %q %q, wanted %q %q", got1, got2, "command", "")
	}

	got1, got2 = getCommandAndArg("MusicButt", "MusicButt", &Config)
	if got1 != "" || got2 != "" {
		t.Errorf("got %q %q, wanted %q %q", got1, got2, "", "")
	}

	got1, got2 = getCommandAndArg("!command argument", "MusicButt", &Config)
	if got1 != "command" || got2 != "argument" {
		t.Errorf("got %q %q, wanted %q %q", got1, got2, "command", "argument")
	}
	got1, got2 = getCommandAndArg("!command", "MusicButt", &Config)
	if got1 != "command" || got2 != "" {
		t.Errorf("got %q %q, wanted %q %q", got1, got2, "command", "")
	}

	got1, got2 = getCommandAndArg("!", "MusicButt", &Config)
	if got1 != "" || got2 != "" {
		t.Errorf("got %q %q, wanted %q %q", got1, got2, "", "")
	}

	got1, got2 = getCommandAndArg("command argument", "MusicButt", &Config)
	if got1 != "command" || got2 != "argument" {
		t.Errorf("got %q %q, wanted %q %q", got1, got2, "command", "argument")
	}

	got1, got2 = getCommandAndArg("command", "MusicButt", &Config)
	if got1 != "command" || got2 != "" {
		t.Errorf("got %q %q, wanted %q %q", got1, got2, "command", "")
	}

	got1, got2 = getCommandAndArg("", "MusicButt", &Config)
	if got1 != "" || got2 != "" {
		t.Errorf("got %q %q, wanted %q %q", got1, got2, "", "")
	}
}
