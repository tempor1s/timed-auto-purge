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
	Token    string `json:"token"`    // bot token
	OwnerID  string `json:"owner_id"` // your discord id
	Prefix   string `json:"prefix"`   // bot prefix for purge command
	Channel  string `json:"channel"`  // channel to auto purge
	Interval string `json:"interval"` // interval to purge at (24h, 10m, 5s) etc
}

// loadConfig will load the bot config from a simple json file
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
	// get the config
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
		// only execute commands by the "owner" that is set in json config
		if m.Author.ID != config.OwnerID {
			return
		}
		// clear last 100 messages command
		if m.Content == config.Prefix+"cc" {
			rawMsgs, err := session.ChannelMessages(config.Channel, 100, "", "", "")
			if err != nil {
				log.Fatal("error getting last 100 messages:", err)
			}

			var msgs []string
			// get message ids of new messages
			for _, msg := range rawMsgs {
				now := time.Now()
				// get message created at
				created, err := msg.Timestamp.Parse()
				if err != nil {
					log.Println("error parsing timestamp:", err)
					continue
				}
				// ignore messages older than 2 weeks
				if now.Sub(created) < time.Hour*24*14 {
					msgs = append(msgs, msg.ID)
				} // 2 weeks
			}

			// bulk delete messages
			err = session.ChannelMessagesBulkDelete(config.Channel, msgs)
			if err != nil {
				log.Fatal("error deleting messages: ", err)
			}
		}
	})
	// start the discordgo session
	session.Open()
	// create a cron job manager
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

	// run forever until a term event happens
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	session.Close()
}
