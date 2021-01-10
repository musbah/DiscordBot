package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/file"
	"go.uber.org/zap"
)

var conf = koanf.New(".")
var dbPool *pgxpool.Pool
var log *zap.SugaredLogger

func init() {
	err := conf.Load(file.Provider("config.json"), json.Parser())
	if err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	dbPool, err = pgxpool.Connect(context.Background(), conf.String("Database.url"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
}

func main() {

	logger, _ := zap.NewProduction()
	defer logger.Sync()
	log = logger.Sugar()

	discord, err := discordgo.New("Bot " + conf.String("General.token"))
	if err != nil {
		log.Fatalf("Could not initialize bot, %s", err)
	}

	// Register callback events
	discord.AddHandler(guildCreate)
	discord.AddHandler(messageCreate)
	discord.AddHandler(guildMemberAdd)

	// Required bot intents
	discord.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildPresences | discordgo.IntentsGuildMembers)

	// Open a websocket connection to Discord and begin listening.
	err = discord.Open()
	if err != nil {
		log.Fatalf("Error opening connection, %s", err)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	discord.Close()
	dbPool.Close()
}

// Run on guild availability
func guildCreate(session *discordgo.Session, event *discordgo.GuildCreate) {

	newUsers, err := lookUpNewUsers(event.Guild.Members, session.State.User.ID)
	if err != nil {
		log.Errorf("Could not lookup new users, %s", err)
		return
	}

	if len(newUsers) > 0 {
		err = addUsersToDB(newUsers)
		if err != nil {
			log.Errorf("Could not add new users to db, %s", err)
			return
		}
	}
}

func guildMemberAdd(session *discordgo.Session, event *discordgo.GuildMemberAdd) {
	exists, err := doesUserExistInDB(event.User.ID)
	if err != nil {
		log.Errorf("Error checking the db for user existance, %s \n", err)
		return
	}

	if !exists {
		err = addUsersToDB([]user{createDefaultUserStruct(event.User.ID)})
		if err != nil {
			log.Errorf("Could not add a new user to the db, %s", err)
			return
		}
	}
}

func messageCreate(session *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	if m.Author.ID == session.State.User.ID {
		return
	}

	switch m.Content {
	case "!status":
		user, err := getUserStatus(m.Author.ID)
		if err != nil {
			session.ChannelMessageSend(m.ChannelID, "Error getting user information")
			log.Errorf("Error getting user information: %s", err)
			return
		}

		_, err = session.ChannelMessageSend(m.ChannelID, m.Author.Username+"'s stats\n"+user.String())
		if err != nil {
			log.Errorf("Could not send message, %s", err)
			return
		}

	case "!levelup":
		err := levelup(m.Author.ID)
		if err != nil {
			session.ChannelMessageSend(m.ChannelID, "Error leveling up")
			log.Errorf("Error leveling up: %s", err)
			return
		}

		session.ChannelMessageSend(m.ChannelID, m.Author.Username+" has sucessfully leveled up")
	default:
	}

}
