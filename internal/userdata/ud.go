// Package userdata parses arguments and config files.
// Return structs with parsed data.
// Return error if something went wrong or data missed.
package userdata

import (
	"fmt"
	"os"
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

	if len(args) >= 3 {
		ud.Pass = args[0]
		ud.Nick = args[1]
		ud.Join = args[2]
		return &ud, nil
	}

	configPath := args[0]
	var sections []gScan.Data
	if _, err := os.Stat(configPath); err == nil {
		sections, err = gurlf.ScanFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("%s: scan config: %v", op, err)
		}
	} else {
		configData := unsafe.Slice(unsafe.StringData(configPath), len(configPath))
		sections, err = gurlf.Scan(configData)
		if err != nil {
			return nil, fmt.Errorf("%s: scan args: %v", op, err)
		}
	}

	if err := gurlf.Unmarshal(sections[0], &ud); err != nil {
		return nil, fmt.Errorf("%s: unmarshal: %v", op, err)
	}
	for _, arg := range args {
		idx := strings.Index(arg, "-j=")
		if idx != -1 {
			idx = strings.Index(arg, "join=")
			if idx != -1 {
				continue
			}
		}
		ud.Join = arg[idx+3:]
	}

	errMsg := ""
	if ud.Pass == "" {
		errMsg += "missing PASS "
	}
	if ud.Nick == "" {
		errMsg += "missing NICK "
	}
	if ud.Join == "" {
		errMsg += "missing JOIN "
	}
	if errMsg != "" {
		return nil, fmt.Errorf("%s: missing arguments: %s", op, errMsg)
	}

	return &ud, nil
}
