package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var BotId string
var PushoverToken string
var StuartUserKey string
var UserCache map[string]int64

type PushoverBody struct {
	Token    string `json:"token"`
	User     string `json:"user"`
	Message  string `json:"message"`
	Priority int    `json:"priority"`
	Sound    string `json:"sound"`
	URL      string `json:"url"`
	UrlTitle string `json:"url_title"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file")
	}

	UserCache = make(map[string]int64)
	PushoverToken = os.Getenv("PUSHOVER_API_TOKEN")
	StuartUserKey = os.Getenv("PUSHOVER_USER_KEY")

	goBot, err := discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))
	if err != nil {
		panic(err)
	}

	goBot.Identify.Intents = discordgo.IntentsDirectMessages

	u, err := goBot.User("@me")
	if err != nil {
		panic(err)
	}
	BotId = u.ID

	goBot.AddHandler(ready)
	goBot.AddHandler(handleMessage)

	err = goBot.Open()
	if err != nil {
		panic(err)
	}
	log.Printf("Bot opened!")
	defer goBot.Close()
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	log.Printf("Bot closed!")
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateGameStatus(0, "Accepting hug requests, DM me for more details!")
}

func SendHugRequest(username string, discordID string) {
	log.Printf("Hug request from %s", username)
	pushoverBody := PushoverBody{
		Token:    PushoverToken,
		User:     StuartUserKey,
		Message:  fmt.Sprintf("A hug was requested from %s", username),
		Priority: 0,
		Sound:    "bingbong",
		URL:      fmt.Sprintf("https://discord.com/channels/@me/%v", discordID),
		UrlTitle: "Send hug",
	}
	jsonBody, err := json.Marshal(pushoverBody)
	if err != nil {
		log.Fatal("Failed to marshal body")
	}
	resp, err := http.Post("https://api.pushover.net/1/messages.json", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Fatalf("Failed to send pushover request: %v", err)
		return
	}
	defer resp.Body.Close()
}

func handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == BotId {
		return
	}
	if m.Content == "hug" {
		discordID := m.Author.ID
		if val, ok := UserCache[discordID]; ok {
			if time.Now().Before(time.Unix(val, 0).Add(24 * time.Hour)) {
				s.ChannelMessageSendReply(m.ChannelID, "It has not been 24 hours since your last hug request!", m.Reference())
				return
			}
		}
		UserCache[discordID] = time.Now().Unix()
		SendHugRequest(m.Author.Username, discordID)
		s.ChannelMessageSendReply(m.ChannelID, "Your hug request has been sent!", m.Reference())
	} else {
		s.ChannelMessageSendReply(m.ChannelID, "Please send me `hug` to request a hug. Note, this can be only done once every 24 hours.", m.Reference())
	}
}
