package telegram

import (
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		input    string
		wantType CommandType
		wantName string
	}{
		// List commands
		{"/list", CmdList, ""},
		{"/status", CmdList, ""},

		// Single container start
		{"/start mycontainer", CmdStart, "mycontainer"},
		{"/start nginx", CmdStart, "nginx"},

		// Single container stop
		{"/stop mycontainer", CmdStop, "mycontainer"},
		{"/stop postgres", CmdStop, "postgres"},

		// Single container restart
		{"/restart mycontainer", CmdRestart, "mycontainer"},
		{"/restart web", CmdRestart, "web"},

		// Bulk operations
		{"/start all", CmdStartAll, ""},
		{"/stop all", CmdStopAll, ""},
		{"/restart all", CmdRestartAll, ""},

		// Group operations
		{"/start group media", CmdStartGroup, "media"},
		{"/stop group db", CmdStopGroup, "db"},
		{"/start group services", CmdStartGroup, "services"},

		// Unknown commands
		{"/unknown", CmdUnknown, ""},
		{"", CmdUnknown, ""},
		{"/start", CmdUnknown, ""},
		{"/stop group", CmdUnknown, ""},
		{"hello world", CmdUnknown, ""},

		// Whitespace handling
		{" /list ", CmdList, ""},
		{"/start   container", CmdStart, "container"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			cmd := Parse(tc.input)
			if cmd.Type != tc.wantType {
				t.Errorf("Parse(%q).Type = %d, want %d", tc.input, cmd.Type, tc.wantType)
			}
			if cmd.Name != tc.wantName {
				t.Errorf("Parse(%q).Name = %q, want %q", tc.input, cmd.Name, tc.wantName)
			}
		})
	}
}
