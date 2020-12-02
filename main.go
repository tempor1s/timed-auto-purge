package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

// Config represents the json config
type Config struct {
	Token    string `json:"token"`
	OwnerID  string `json:"owner_id"`
	Prefix   string `json:"prefix"`
	Channel  string `json:"channel"`
	Interval string `json:"interval"`
}

func loadConfig() Config {
	var config Config
	configFile, err := os.Open("config.json")
	defer configFile.Close()
	if err != nil {
		log.Fatal(err)
	}
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)
	return config
}

func main() {
	config := loadConfig()

	// start the discord bot
	session, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		log.Fatal("failed to create discord client:", err)
	}

	// add a basic message handler for a simple clear command
	session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}

		if m.Author.ID != "190295254073606144" {
			return
		}

		if m.Content == config.Prefix+"cc" {
			rawMsgs, err := session.ChannelMessages(config.Channel, 100, "", "", "")
			if err != nil {
				log.Fatal("error getting last 100 messages:", err)
			}

			var msgs []string
			for _, msg := range rawMsgs {
				now := time.Now()
				created, err := msg.Timestamp.Parse()
				if err != nil {
					log.Println("error parsing timestamp:", err)
					continue
				}

				if now.Sub(created) < time.Hour*24*14 {
					msgs = append(msgs, msg.ID)
				} // 2 weeks
			}

			err = session.ChannelMessagesBulkDelete(config.Channel, msgs)
			if err != nil {
				log.Fatal("error deleting messages: ", err)
			}
		}
	})
	// start the discordgo session
	session.Open()
	c := cron.New()
	// runs the cron every x amount of time
	c.AddFunc(fmt.Sprintf("@every %s", config.Interval), func() {
		log.Println("purging channel...")
		rawMsgs, err := session.ChannelMessages(config.Channel, 100, "", "", "")
		if err != nil {
			log.Fatal("error getting last 100 messages:", err)
		}

		var msgs []string
		for _, msg := range rawMsgs {
			now := time.Now()
			created, err := msg.Timestamp.Parse()
			if err != nil {
				log.Println("error parsing timestamp:", err)
				continue
			}

			if now.Sub(created) < time.Hour*24*14 {
				msgs = append(msgs, msg.ID)
			} // 2 weeks
		}

		err = session.ChannelMessagesBulkDelete(config.Channel, msgs)
		if err != nil {
			log.Fatal("error deleting messages: ", err)
		}
	})
	// start the cron jobs
	c.Start()

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	session.Close()
}
