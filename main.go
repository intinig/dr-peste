package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/intinig/dr-peste/commands"
	"github.com/intinig/dr-peste/db"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file:", err)
	}

	// Get Discord token from environment variables
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("No Discord token provided. Set the DISCORD_TOKEN environment variable.")
	}

	// Get Guild ID from environment variables (optional)
	guildID := os.Getenv("GUILD_ID")
	if guildID == "" {
		log.Println("No Guild ID provided. Commands will be registered globally (can take up to an hour).")
		log.Println("To register commands instantly for a specific server, set the GUILD_ID environment variable.")
	} else {
		log.Printf("Guild ID provided: %s. Commands will be registered for this server only.", guildID)
	}

	// Initialize the database
	if err := db.Initialize(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Create a new Discord session
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Error creating Discord session:", err)
	}

	// Register slash command handler
	commands.RegisterSlashCommands(dg)

	// Open a websocket connection to Discord
	err = dg.Open()
	if err != nil {
		log.Fatal("Error opening connection:", err)
	}
	defer dg.Close()

	// Register slash commands with Discord
	log.Println("Registering slash commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands.SlashCommands()))
	for i, cmd := range commands.SlashCommands() {
		rcmd, err := dg.ApplicationCommandCreate(dg.State.User.ID, guildID, cmd)
		if err != nil {
			log.Printf("Error creating '%s' command: %v", cmd.Name, err)
		} else {
			registeredCommands[i] = rcmd
			log.Printf("Registered command: %s", cmd.Name)
		}
	}

	// Log that the bot is running
	log.Println("Docteur Peste is now running. Press CTRL-C to exit.")
	
	// Wait for a CTRL-C or other termination signal
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Clean up slash commands on exit
	log.Println("Removing slash commands...")
	for _, cmd := range registeredCommands {
		err := dg.ApplicationCommandDelete(dg.State.User.ID, guildID, cmd.ID)
		if err != nil {
			log.Printf("Error removing '%s' command: %v", cmd.Name, err)
		}
	}
} 