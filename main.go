// Package main do nothing
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"unsafe"

	"github.com/gorilla/websocket"
)

func main() {
	const op = "main.main"

	url := "wss://irc-ws.chat.twitch.tv:443"
	dialer := websocket.DefaultDialer
	conn, resp, err := dialer.Dial(url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: error dialing: %v", op, err)
		return
	}
	defer conn.Close()

	if resp.StatusCode != 101 {
		fmt.Fprintf(os.Stderr, "%s: unexpected status code: %v", op, resp.StatusCode)
		return
	}

	fmt.Println("Connected to the server")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errChan := make(chan error)

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
				fmt.Println(unsafe.String(unsafe.SliceData(message), len(message)))
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
		fmt.Fprintln(os.Stderr, err)
		return
	case <-ctx.Done():
		fmt.Println("Connection closed")
	}
}
