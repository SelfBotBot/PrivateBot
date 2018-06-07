package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/SelfBotBot/PrivateBot"
)

func main() {

	waitingRooms := PrivateBot.DefaultConfig
	e(waitingRooms.Load())

	bot, err := PrivateBot.New(waitingRooms)
	e(err)

	err = bot.Session.Open()
	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
		panic(err)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("PrivacyBot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	bot.Session.Close()

}

func e(err error) {
	if err != nil {
		panic(err)
	}
}
