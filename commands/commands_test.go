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
	trueFalse := [2]bool{true, false}
	for _, truth := range trueFalse {
		// message with nickname as prefix
		got1, got2 := getCommandAndArg("MusicButt command argument", "MusicButt", truth, &Config)
		if got1 != "command" || got2 != "argument" {
			t.Errorf("got %q %q  wanted %q %q, truth %t", got1, got2, "command", "argument", truth)
		}

		got1, got2 = getCommandAndArg("MusicButt command", "MusicButt", truth, &Config)
		if got1 != "command" || got2 != "" {
			t.Errorf("got %q %q, wanted %q %q, truth %t", got1, got2, "command", "", truth)
		}

		got1, got2 = getCommandAndArg("MusicButt", "MusicButt", truth, &Config)
		if got1 != "" || got2 != "" {
			t.Errorf("got %q %q, wanted %q %q, truth: %t", got1, got2, "", "", truth)
		}

		got1, got2 = getCommandAndArg("!command argument", "MusicButt", truth, &Config)
		if got1 != "command" || got2 != "argument" {
			t.Errorf("got %q %q, wanted %q %q, truth: %t", got1, got2, "command", "argument", truth)
		}
		got1, got2 = getCommandAndArg("!command", "MusicButt", truth, &Config)
		if got1 != "command" || got2 != "" {
			t.Errorf("got %q %q, wanted %q %q, truth: %t", got1, got2, "command", "", truth)
		}

		got1, got2 = getCommandAndArg("!", "MusicButt", truth, &Config)
		if got1 != "" || got2 != "" {
			t.Errorf("got %q %q, wanted %q %q, truth: %t", got1, got2, "", "", truth)
		}

	}

	got1, got2 := getCommandAndArg("command argument", "MusicButt", true, &Config)
	if got1 != "command" || got2 != "argument" {
		t.Errorf("got %q %q, wanted %q %q", got1, got2, "command", "argument")
	}

	got1, got2 = getCommandAndArg("command", "MusicButt", true, &Config)
	if got1 != "command" || got2 != "" {
		t.Errorf("got %q %q, wanted %q %q", got1, got2, "command", "")
	}

	got1, got2 = getCommandAndArg("", "MusicButt", true, &Config)
	if got1 != "" || got2 != "" {
		t.Errorf("got %q %q, wanted %q %q", got1, got2, "", "")
	}
}
