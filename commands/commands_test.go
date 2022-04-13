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
		if got1 != "command" {
			t.Errorf("got %q, wanted %q, truth %t", got1, "command", truth)
		}
		if got2 != "argument" {
			t.Errorf("got %q, wanted %q, truth %t", got2, "argument", truth)
		}

		got1, got2 = getCommandAndArg("MusicButt command", "MusicButt", truth, &Config)
		if got1 != "command" {
			t.Errorf("got %q, wanted %q, truth %t", got1, "command", truth)
		}
		if got2 != "" {
			t.Errorf("got %q, wanted %q, truth %t", got2, "", truth)
		}

		got1, got2 = getCommandAndArg("MusicButt", "MusicButt", truth, &Config)
		if got1 != "" {
			t.Errorf("got %q, wanted %q, truth: %t", got1, "", truth)
		}
		if got2 != "" {
			t.Errorf("got %q, wanted %q, truth: %t", got2, "", truth)
		}

		got1, got2 = getCommandAndArg("!command argument", "MusicButt", truth, &Config)
		if got1 != "command" {
			t.Errorf("got %q, wanted %q, truth: %t", got1, "command", truth)
		}
		if got2 != "argument" {
			t.Errorf("got %q, wanted %q, truth: %t", got2, "argument", truth)
		}
		got1, got2 = getCommandAndArg("!command", "MusicButt", truth, &Config)
		if got1 != "command" {
			t.Errorf("got %q, wanted %q, truth: %t", got1, "command", truth)
		}
		if got2 != "" {
			t.Errorf("got %q, wanted %q, truth: %t", got2, "", truth)
		}

		got1, got2 = getCommandAndArg("!", "MusicButt", truth, &Config)
		if got1 != "" {
			t.Errorf("got %q, wanted %q, truth: %t", got1, "", truth)
		}
		if got2 != "" {
			t.Errorf("got %q, wanted %q, truth: %t", got2, "", truth)
		}

	}

	got1, got2 := getCommandAndArg("command argument", "MusicButt", true, &Config)
	if got1 != "command" {
		t.Errorf("got %q, wanted %q", got1, "command")
	}
	if got2 != "argument" {
		t.Errorf("got %q, wanted %q", got2, "argument")
	}

	got1, got2 = getCommandAndArg("command", "MusicButt", true, &Config)
	if got1 != "command" {
		t.Errorf("got %q, wanted %q", got1, "command")
	}
	if got2 != "" {
		t.Errorf("got %q, wanted %q", got2, "")
	}

	got1, got2 = getCommandAndArg("", "MusicButt", true, &Config)
	if got1 != "" {
		t.Errorf("got %q, wanted %q", got1, "")
	}
	if got2 != "" {
		t.Errorf("got %q, wanted %q", got2, "")
	}
}
