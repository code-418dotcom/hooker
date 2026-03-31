package telegram

import (
	"strings"
)

// CommandType is the type of command parsed from a Telegram message.
type CommandType int

const (
	CmdUnknown CommandType = iota
	CmdList
	CmdStart
	CmdStop
	CmdRestart
	CmdStartAll
	CmdStopAll
	CmdRestartAll
	CmdStartGroup
	CmdStopGroup
)

// Command represents a parsed Telegram command.
type Command struct {
	Type CommandType
	Name string // container name or group tag
}

// Parse parses a message text into a Command.
// It handles commands like /list, /start, /stop, /restart with various arguments.
func Parse(text string) Command {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return Command{Type: CmdUnknown}
	}

	switch fields[0] {
	case "/status", "/list":
		return Command{Type: CmdList}
	case "/start":
		return parseStartStop(CmdStart, CmdStartAll, CmdStartGroup, fields[1:])
	case "/stop":
		return parseStartStop(CmdStop, CmdStopAll, CmdStopGroup, fields[1:])
	case "/restart":
		return parseRestart(fields[1:])
	}

	return Command{Type: CmdUnknown}
}

// parseStartStop parses start/stop/restart with optional "all" or "group <tag>" argument.
func parseStartStop(single, all, group CommandType, args []string) Command {
	if len(args) == 0 {
		return Command{Type: CmdUnknown}
	}
	if args[0] == "all" {
		return Command{Type: all}
	}
	if args[0] == "group" {
		if len(args) >= 2 {
			return Command{Type: group, Name: args[1]}
		}
		// Incomplete group command
		return Command{Type: CmdUnknown}
	}
	return Command{Type: single, Name: args[0]}
}

// parseRestart parses restart with optional "all" argument.
func parseRestart(args []string) Command {
	if len(args) == 0 {
		return Command{Type: CmdUnknown}
	}
	if args[0] == "all" {
		return Command{Type: CmdRestartAll}
	}
	return Command{Type: CmdRestart, Name: args[0]}
}
