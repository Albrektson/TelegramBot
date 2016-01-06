package main

import (
	//"fmt"
	"bitbucket.org/albrektson/tbotapi"
	"bufio"
	"bytes"
	"os/exec"
	"strings"
	"time"
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
var channel int = <chat id>
var lastStatus int

func main() {
	for {
		status := portcheck(port, server)
		if status != lastStatus {
			lastStatus = status
			talk(status)
		}
		time.Sleep(5 * time.Minute)
	}
}

//figures out what to say and send it to chat
func talk(status int) {
	bot, err := tbotapi.New(authkey)
	if err != nil {
		return
	}

	switch status {
	case UP:
		send(bot, channel, "Server looks fine now.")

	case DOWN:
		send(bot, channel, "Oh no, the server seems to be down!")

	case BLOCK:
		send(bot, channel, "That's odd, there's something wrong here..")

	case FAIL:
		send(bot, channel, "I've gone tits up! Someone call a doctor?")
	}
}

//send a message to a chat
func send(bot *tbotapi.TelegramBotAPI, chat int, message string) {
	bot.SendMessage(chat, message)
}

func portcheck(port string, server string) int {
	//run nmap and collect results
	executable := exec.Command("nmap", server, "-Pn", "-p", port)
	output, err := executable.Output()
	if err != nil {
		//fmt.Println("failed to run nmap")
		return FAIL
	}

	//parse the output from nmap
	serverFound := false
	reader := bytes.NewReader(output)
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {

		line := scanner.Text()
		if strings.Contains(line, port) {
			serverFound = true
			if strings.Contains(line, "open") {
				//fmt.Println("Server up")
				return UP
			} else if strings.Contains(line, "filtered") {
				//fmt.Println("Port blocked")
				return BLOCK
			}
		}
	}

	if !serverFound {
		//fmt.Println("Server down")
		return DOWN
	}

	return FAIL
}
