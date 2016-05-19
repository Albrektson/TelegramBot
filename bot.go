package main

import (
	"fmt"
	"bitbucket.org/mrd0ll4r/tbotapi"
	"bitbucket.org/mrd0ll4r/tbotapi/model"
	"net"
	"strings"
	"encoding/binary"
	"bytes"
	"time"
	"strconv"
)

const (
	UP = iota
	DOWN
	BLOCK
	FAIL
)

var authkey string = <bot authkey>
var server string = <server address>
var port string = <server port to check>
var defaultChannel int = <chat id>
var lastStatus int

type Stats struct {
	version int
	ident int
	usercount int
	maxusers int
	bandwidthlimit int
}

func main() {
	bot, err := tbotapi.New(authkey)
	if err != nil {
		fmt.Println("Bot connection failed.")
		fmt.Println(err)
		return
	}
	
	go listen(bot)
	for {
		if defaultChannel != 0 {
			status, data := ping(server, port)
			if status != lastStatus {
				lastStatus = status
				talk(bot, defaultChannel, status, data)
			}
		}
		time.Sleep(5 * time.Minute)
	}
}

func listen(bot *tbotapi.TelegramBotAPI) {
	for {
		select {
		case update := <-bot.Updates:	//we should check timestaps for old backlogged requests
			mType := update.Message.Type()
			msg := update.Message
			text := *msg.Text
			channel := msg.Chat.ID
			if mType == model.TextType {
				if text == "/status"{
					status, data := ping(server, port)
					talk(bot, channel, status, data)
				} else if text == "/start" {
					send(bot, channel, "Hello.")
				} else {
					send(bot, channel, "I can't let you do that, Dave.")
				}

		case update := <-bot.Errors:
			fmt.Printf("Err: %s\n", update)
		}
	}
}

//send appropriate message to chat
func talk(bot *tbotapi.TelegramBotAPI, channel int, status int, data Stats) {
	switch status {
	case UP:
		var buffer bytes.Buffer
		buffer.WriteString("Server appears to be up.\n")
		buffer.WriteString("Current Mumble users: ")
		buffer.WriteString(strconv.Itoa(data.usercount))
		buffer.WriteString("/")
		buffer.WriteString(strconv.Itoa(data.maxusers))
		output := buffer.String()
		send(bot, channel, output)

	case DOWN:
		send(bot, channel, "I can't reach the server's network at all.")
		
	case BLOCK:
		send(bot, channel, "I can reach the server, but not mumble!")

	case FAIL:
		send(bot, channel, "I've gone tits up! Someone call a doctor?")
	}
}

//send a message to a chat
func send(bot *tbotapi.TelegramBotAPI, chat int, message string) {
	bot.SendMessage(chat, message)
}

func ping(server string, port string) (int, Stats) {
	data := new(Stats)
	
	host := strings.Join([]string{server, port}, ":")
	conn, err := net.Dial("udp", host)
	if err != nil {
		fmt.Println("failed to dial server")
		return DOWN, *data
	} else {
		defer conn.Close()		
		conn.SetDeadline(time.Now().Add(time.Second*5))
		
		//craft ident request package
		message := make([]byte, 12)
		curTime := time.Now().UnixNano()
		binary.BigEndian.PutUint64(message[4:], uint64(curTime))
		
		//send request package to server
		n, err := conn.Write(message)
		if err != nil {
			fmt.Println(err)
			return BLOCK, *data
		}
		fmt.Printf("bytes sent: %v\n",n)
		
		//get information package from server
		ans := make([]byte, 24)
		s, err := conn.Read(ans)
		if err != nil {
			fmt.Println(err)
			return BLOCK, *data
		}
		
		//parse information and return
		fmt.Printf("bytes read: %v\n",s)
		if (s > 0) { 
			data.version = int(binary.BigEndian.Uint32(ans[:4]))	//this mangles the version data, it's not actually encoded
			data.ident = int(binary.BigEndian.Uint64(ans[4:12]))
			data.usercount = int(binary.BigEndian.Uint32(ans[12:16]))
			data.maxusers = int(binary.BigEndian.Uint32(ans[16:20]))
			data.bandwidthlimit = int(binary.BigEndian.Uint32(ans[20:]))
			
			return UP, *data
		}
	}
	return FAIL, *data
}
