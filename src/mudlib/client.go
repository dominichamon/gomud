// Package mudlib is a mud engine.
package mudlib

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

type messageType int

const ( // message types
	messageTypeSay messageType = iota
	messageTypeTell
	messageTypeEmote
	messageTypeShout
	messageTypeJoin
	messageTypeQuit
	messageTypeWho
	messageTypeEnterRoom
	messageTypeLeaveRoom
)

type message struct {
	from        client
	to          string
	message     string
	messageType messageType
}

type client struct {
	conn   net.Conn
	player string
	ch     chan message
	log    *log.Logger
}

func (c client) readLines() {
	bufc := bufio.NewReader(c.conn)
	for {
		line, err := bufc.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		c.log.Printf("command %q: %q.\n", c.player, line)

		parts := strings.Fields(line)
		if err := doCommand(c, parts[0], parts[1:]); err != nil {
			c.log.Printf("%+v during command %q\n", err, line)
		}
	}
}

func sameRoom(c client, msg message) bool {
	player, err := players.get(c.player)
	if err != nil {
		errorLog.Fatalf("%+v", err)
		return false
	}
	fromPlayer, err := players.get(msg.from.player)
	if err != nil {
		errorLog.Fatalf("%+v", err)
		return false
	}
	return player.room == fromPlayer.room
}

func (c client) writeLinesFrom(ch <-chan message) {
	for msg := range ch {
		from := msg.from.player
		toPrint := ""
		// TODO: Register command per message type for colors/format string and location restriction
		switch msg.messageType {
		case messageTypeSay:
			if sameRoom(c, msg) {
				if msg.from == c {
					toPrint = setFg(colorYellow, fmt.Sprintf("You say \"%s\".", msg.message))
				} else {
					toPrint = setFg(colorYellow, fmt.Sprintf("%s says \"%s\".", from, msg.message))
				}
			}
		case messageTypeTell:
			if msg.from == c {
				toPrint = setFg(colorGreen, fmt.Sprintf("You tell %s \"%s\".", msg.to, msg.message))
			} else if msg.to == c.player {
				toPrint = setFg(colorGreen, fmt.Sprintf("%s tells you \"%s\".", from, msg.message))
			}
		case messageTypeEmote:
			if sameRoom(c, msg) {
				if msg.from == c {
					// TODO: self-emote is tricky: "/me prances" -> "xxx prances" or "You prance"
				} else {
					toPrint = setFg(colorMagenta, fmt.Sprintf("%s %s.", from, msg.message))
				}
			}
		case messageTypeShout:
			if msg.from == c {
				toPrint = setFgBold(colorCyan, fmt.Sprintf("You shout \"%s\".", msg.message))
			} else {
				toPrint = setFgBold(colorCyan, fmt.Sprintf("%s shouts \"%s\".", from, msg.message))
			}
		case messageTypeQuit:
			if sameRoom(c, msg) {
				if msg.from == c {
					toPrint = setFgBold(colorRed, fmt.Sprintf("You have quit."))
				} else {
					toPrint = setFgBold(colorRed, fmt.Sprintf("%s has quit.", from))
				}
			}
		case messageTypeJoin:
			if sameRoom(c, msg) {
				if msg.from == c {
					toPrint = setFgBold(colorRed, fmt.Sprintf("You have joined."))
				} else {
					toPrint = setFgBold(colorRed, fmt.Sprintf("%s has joined.", from))
				}
			}
		case messageTypeEnterRoom:
			if sameRoom(c, msg) && msg.from != c {
				toPrint = setFg(colorCyan, fmt.Sprintf("%s enters.", from))
			}
		case messageTypeLeaveRoom:
			player, err := players.get(c.player)
			if err != nil {
				errorLog.Fatalf("%+v", err)
				continue
			}
			if player.room == msg.message && msg.from != c {
				toPrint = setFg(colorCyan, fmt.Sprintf("%s leaves.", from))
			}
		default:
			c.log.Printf("Unhandled message type: %+v", msg)
			errorLog.Printf("Unhandled message type: %q %+v", c.player, msg)
			continue
		}
		if len(toPrint) == 0 {
			continue
		}
		_, err := io.WriteString(c.conn, toPrint+"\n")
		if err != nil {
			errorLog.Printf("Error writing '%q'\n", toPrint)
		}
	}
}
