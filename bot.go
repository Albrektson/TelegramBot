package main

import (
	"fmt"
	"github.com/mrd0ll4r/tbotapi"
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
var timeZone = "Europe/Stockholm"
var timeFormat = "15:04 on Jan 2"

var lastStatus int
var initTime int
var failedSince time.Time

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
	defer bot.Close()
	
	initTime = int (time.Now().Unix())
	
	go listen(bot)
	for {
		if defaultChannel != 0 {
			status, data := ping(server, port)
			if status != UP {
				if failedSince.IsZero() == true {
					failedSince = time.Now()
				} else if status != lastStatus {
					lastStatus = status
					talk(bot, defaultChannel, status, data)
				}
			} else if status != lastStatus {
				//if the server recovered from a failure, send a message and remove failure timestamp
				lastStatus = status
				failedSince = time.Time{} //zero out value
				talk(bot, defaultChannel, status, data)
			} else {
				//in case of one-time failure, just remove failure timestamp
				failedSince = time.Time{}
			}
		}
		time.Sleep(5 * time.Minute)
	}
}

func listen(bot *tbotapi.TelegramBotAPI) {
	for {
		input := <-bot.Updates
		
		//check for errors
		err := input.Error()
		if err != nil {
			fmt.Printf("Err: %s\n", input)
			continue
		}		
		
		//handle bot update data
		update := input.Update()
		msg := update.Message
		if msg.Date < initTime {
			//backlogged message, cancel read
			continue
		}
		text := *msg.Text
		channel := msg.Chat.ID
		if msg.Type() == tbotapi.TextMessage {
			if text == "/status"{
				status, data := ping(server, port)
				talk(bot, channel, status, data)
			} else if text == "/start" {
				send(bot, channel, "Hello.")
			} else {
				send(bot, channel, "I can't let you do that, Dave.")
			}
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
		send(bot, channel, "I can't reach the server's network at all. Is the DDNS broken?")
		
	case BLOCK:
		var buffer bytes.Buffer
		buffer.WriteString("I can't reach the server!\n")
		buffer.WriteString("Server unreachable since ")
		buffer.WriteString(parseTime(failedSince))
		output := buffer.String()
		send(bot, channel, output)

	case FAIL:
		send(bot, channel, "I've gone tits up! Someone call a doctor?")
	}
}

func parseTime(t time.Time) string {
	loc, _ := time.LoadLocation(timeZone)
	parsedTime := t.In(loc).Format(timeFormat)
	return parsedTime
}

//send a message to a chat
func send(bot *tbotapi.TelegramBotAPI, chat int, text string) {
	recipient := tbotapi.NewChatRecipient(chat)
	message := bot.NewOutgoingMessage(recipient, markdown(text))
	message = message.SetMarkdown(true)
	message.Send()
}

//convert text to fixedsys markdown text
func markdown(text string) string {
	var buffer bytes.Buffer
	buffer.WriteString("`")
	buffer.WriteString(text)
	buffer.WriteString("`")
	return buffer.String()
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
