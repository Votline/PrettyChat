// Package main is a simple IRC client for Twitch chat.
package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"

	"prcht/internal/authserver"
	"prcht/internal/chat"
	rb "prcht/internal/ringbuffer"
	"prcht/internal/userdata"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const helpMsg = `
Usage (choose your way):
    1. From config file: prcht <config_path> <args>
    2. From config in args: prcht "[config]...[\config]" <args>
    3. From argumens: prcht <PASS> <NICK> <JOIN> <args>
    4. Help message: prcht -h or prcht help

Needed arguments & config fields:
    PASS: Access token of your Twitch account (e.g. oauth:1234abcd)
    NICK: Your Twitch username
    JOIN: Channel name to join (e.g. bratishkinoff)

How to get PASS and NICK:
    1. Run app with args: 'prcht -i <clientID> <clientSecret> <args>'
    2. Open URL from console in browser (if it not opened automatically)
    3. Click 'Allow' button
    4. App will print and save to file your PASS
    5. Next time you can run app with new config file (prcht.gurlf)

Other arguments:
    -h/help: Show help message
    -d/debug: Enable debug mode
    -a/auth: Start HTTP server for exchanging code to tokens
    -j=/join=: Dynamically change 'JOIN' argument. Usage: 'prcht cfg.gurlf -j=bratishkinoff'
`

func main() {
	const op = "main.main"

	go func() {
		fmt.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	args := os.Args[1:]
	if args[0] == "help" || args[0] == "-h" {
		fmt.Print(helpMsg)
		return
	}

	ud, err := userdata.Parse(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: error parsing args: %v\n", op, err)
		return
	}
	log := initLog(ud.Debug)

	if ud.Auth {
		authserver.StartServer(ud.ClientID, ud.ClientSecret, log)
		return
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
		fmt.Fprintf(os.Stderr, "%s: error parsing args: %s\n", op, errMsg)
		return
	}

	log.Debug("User data",
		zap.String("PASS", ud.Pass),
		zap.String("NICK", ud.Nick),
		zap.String("JOIN", ud.Join))

	conn, err := establishConnect()
	if err != nil {
		log.Fatal("error establishing connection",
			zap.String("op", op),
			zap.Error(err))
	}
	defer conn.Close()
	log.Debug("Successfully connected")

	if err := sendUserData(conn, ud); err != nil {
		log.Fatal("send user data",
			zap.String("op", op),
			zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errChan := make(chan error, 2)

	chatMsg := chat.ChatMessage{
		Nick:   new(string),
		Msg:    new(string),
		Join:   []byte("#" + ud.Join + " :"),
		Status: rb.NewRB[string](10),
	}
	users := make(map[string]*chat.User)
	go func() {
		defer close(errChan)
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, message, err := conn.ReadMessage()
				if err != nil {
					err = fmt.Errorf("%s: error reading message: %v", op, err)
					errChan <- err
					return
				}

				if bytes.HasPrefix(message, []byte("PING")) {
					if err := conn.WriteMessage(websocket.TextMessage, []byte("PONG")); err != nil {
						err = fmt.Errorf("%s: error writing message: %v", op, err)
						errChan <- err
						return
					}
					continue
				}

				if err := chat.PrintMessage(&users, &chatMsg, message); err != nil {
					log.Error("error extarct message",
						zap.String("op", op),
						zap.String("msg", string(message)),
						zap.Error(err))
					continue
				}
				*chatMsg.Nick = ""
				*chatMsg.Msg = ""
				chatMsg.Status.Reset()
			}
		}
	}()

	go func() {
		defer close(errChan)
		defer cancel()

		stdinScanner := bufio.NewScanner(os.Stdin)
		select {
		case <-ctx.Done():
			return
		default:
			for stdinScanner.Scan() {
				err := conn.WriteMessage(websocket.TextMessage, stdinScanner.Bytes())
				if err != nil {
					err = fmt.Errorf("%s: error writing message: %v", op, err)
					errChan <- err
					return
				}
			}
			if err := stdinScanner.Err(); err != nil {
				err = fmt.Errorf("%s: error reading from stdin: %v", op, err)
				errChan <- err
				return
			}
		}
	}()

	select {
	case err := <-errChan:
		log.Fatal("error",
			zap.String("op", op),
			zap.Error(err))
	case <-ctx.Done():
		log.Debug("Successfully disconnected")
	}
}

func establishConnect() (*websocket.Conn, error) {
	const op = "main.establishConnect"

	url := "wss://irc-ws.chat.twitch.tv:443"
	dialer := websocket.DefaultDialer
	conn, resp, err := dialer.Dial(url, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: error dialing: %v", op, err)
	}

	if resp.StatusCode != 101 {
		return nil, fmt.Errorf("%s: unexpected status code: %v", op, resp.StatusCode)
	}

	return conn, nil
}

func sendUserData(conn *websocket.Conn, ud *userdata.UserData) error {
	const op = "main.sendUserData"

	if err := conn.WriteMessage(websocket.TextMessage, []byte("CAP REQ :twitch.tv/tags twitch.tv/commands")); err != nil {
		return fmt.Errorf("%s: error requesting tags: %v", op, err)
	}

	passCmd := "PASS oauth:" + ud.Pass
	if err := conn.WriteMessage(websocket.TextMessage, []byte(passCmd)); err != nil {
		return fmt.Errorf("%s: error writing PASS: %v", op, err)
	}

	nickCmd := "NICK " + ud.Nick
	if err := conn.WriteMessage(websocket.TextMessage, []byte(nickCmd)); err != nil {
		return fmt.Errorf("%s: error writing NICK: %v", op, err)
	}

	joinCmd := "JOIN #" + ud.Join
	if err := conn.WriteMessage(websocket.TextMessage, []byte(joinCmd)); err != nil {
		return fmt.Errorf("%s: error writing JOIN: %v", op, err)
	}

	return nil
}

func initLog(dbg bool) *zap.Logger {
	cfg := zap.NewDevelopmentConfig()
	cfg.Encoding = "console"
	cfg.EncoderConfig.TimeKey = ""
	cfg.DisableStacktrace = true
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.EncoderConfig.ConsoleSeparator = " | "
	cfg.Level.SetLevel(zap.ErrorLevel)

	if dbg {
		cfg.Level.SetLevel(zap.DebugLevel)
	}
	log, _ := cfg.Build()

	return log
}
