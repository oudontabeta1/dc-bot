package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/oudentabetai/dc-bot/discord"
	"github.com/oudentabetai/dc-bot/storage"
)

var (
	GuildID string
	dgs     *discordgo.Session
	version = "dev"
)

func main() {
	sessionManager := &discord.DiscordSessionManager{}
	if err := storage.ConfigMgr.Load(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dgs = sessionManager.InitializeSession(storage.Envs.DISCORD_BOT_TOKEN)
	dgs.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuilds | discordgo.IntentsGuildMembers | discordgo.IntentsAll | discordgo.PermissionSendMessages

	dgs.AddHandler(discord.OnMessageCreate)
	dgs.AddHandler(discord.OnInteractionCreate)
	dgs.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %s", s.State.User.Username)
	})

	if err := dgs.Open(); err != nil {
		log.Fatalf("Failed to open Discord session: %v", err)
	}
	defer dgs.Close()

	sendStartupVersionLog(dgs)
	log.Println("Launched")

	//deleteAllGlobalCommands(dgs, os.Getenv("APPLICATION_ID"))
	discord.SyncCommands(dgs, "", storage.Envs.APPLICATION_ID)
	waitForExitSignal()
}

func waitForExitSignal() {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func sendStartupVersionLog(s *discordgo.Session) {
	if storage.Envs.LOG_CHANNEL_ID == "" {
		return
	}

	msg := "Launched Bot Version is: " + version
	if _, err := s.ChannelMessageSend(storage.Envs.LOG_CHANNEL_ID, msg); err != nil {
		log.Printf("Failed To Send Launch Log: %v", err)
	}
}

func deleteAllGlobalCommands(s *discordgo.Session, appID string) {
	_, err := s.ApplicationCommandBulkOverwrite(appID, "", []*discordgo.ApplicationCommand{})

	if err != nil {
		log.Printf("Failed To Remove Global Commands: %v", err)
		return
	}
	log.Println("Succesfully Removed Global Commands。")
}
