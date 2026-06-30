// Package userdata parses arguments and config files.
// Return structs with parsed data.
// Return error if something went wrong or data missed.
package userdata

import (
	"fmt"
	"slices"
	"strings"
	"unsafe"

	"github.com/Votline/Gurlf"
	gScan "github.com/Votline/Gurlf/pkg/scanner"
)

type UserData struct {
	Pass         string `gurlf:"PASS"`
	Nick         string `gurlf:"NICK"`
	Join         string `gurlf:"JOIN"`
	ClientID     string
	ClientSecret string
	Debug        bool
	Auth         bool
}

func Parse(args []string) (*UserData, error) {
	const op = "userdata.Parse"
	var ud UserData

	if len(args) == 0 {
		return nil, fmt.Errorf("%s: not enough arguments", op)
	}

	dbgMode := slices.Contains(args, "-d") || slices.Contains(args, "debug")
	if dbgMode {
		ud.Debug = true
	}

	authMode := slices.Contains(args, "-a") || slices.Contains(args, "auth")
	if authMode {
		ud.Auth = true
		if len(args) < 3 {
			return nil, fmt.Errorf("%s: missing arguments for auth mode", op)
		}
		ud.ClientID = args[1]
		ud.ClientSecret = args[2]
		return &ud, nil
	}

	configPath := args[0]
	var sections []gScan.Data
	sections, err := gurlf.ScanFile(configPath)
	if err == nil {
		if err = configResult(&ud, sections, args[1:]); err != nil {
			return nil, fmt.Errorf("%s: configResult: %v", op, err)
		}
		return &ud, nil
	}
	configData := unsafe.Slice(unsafe.StringData(configPath), len(configPath))
	sections, err = gurlf.Scan(configData)
	if err == nil {
		if err = configResult(&ud, sections, args[1:]); err != nil {
			return nil, fmt.Errorf("%s: configResult: %v", op, err)
		}
		return &ud, nil
	}

	if len(args) >= 3 {
		ud.Pass = args[0]
		ud.Nick = args[1]
		ud.Join = args[2]
	}

	return &ud, nil
}

func configResult(ud *UserData, sections []gScan.Data, args []string) error {
	const op = "userdata.configResult"

	if len(sections) == 0 {
		return fmt.Errorf("%s: no sections found", op)
	}

	if err := gurlf.Unmarshal(sections[0], ud); err != nil {
		return fmt.Errorf("%s: unmarshal: %v", op, err)
	}
	for _, arg := range args {
		if prefix, ok := strings.CutPrefix(arg, "-j="); ok {
			ud.Join = prefix
			continue
		}
	}
	return nil
}
