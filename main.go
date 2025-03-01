package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/tinaxd/gamebot/gamebot"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	db *gorm.DB
)

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		panic("DISCORD_TOKEN is not set")
	}

	var err error
	db, err = gorm.Open(sqlite.Open("gamebot.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to db")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("failed to create instnace: %v", err)
	}

	dg.AddHandler(handleReady)
	dg.AddHandler(handleInteractionCreate)
	dg.AddHandler(handleButtonInteraction)
	dg.AddHandler(handleGameCommand)

	err = dg.Open()
	if err != nil {
		log.Fatalf("failed to open instance: %v", err)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

func handleReady(dg *discordgo.Session, r *discordgo.Ready) {
	log.Printf("Connected to Discord")

	_, err := dg.ApplicationCommandCreate(dg.State.User.ID, "", appCommand)
	if err != nil {
		log.Printf("Failed to create global application command: %v", err)
	} else {
		log.Printf("Created global application command")
	}
}

func handleInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		// Handle command interactions
		if i.ApplicationCommandData().Name == "game" {
			handleGameCommand(s, i)
		}
	} else if i.Type == discordgo.InteractionMessageComponent {
		// Handle button interactions
		if i.MessageComponentData().ComponentType == discordgo.ButtonComponent {
			handleButtonInteraction(s, i)
		}
	}
}

func handleGameCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options

	// Find options by name
	var gameName, timeStr string
	for _, opt := range options {
		if opt.Name == "game" {
			gameName = opt.StringValue()
		} else if opt.Name == "time" {
			timeStr = opt.StringValue()
		}
	}

	// Validate time string
	datetime, err := gamebot.ParseTargetTimeFormat(timeStr)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Invalid time format. Please use `HH:MM`.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	targetTime := gamebot.CalculateNextDay(time.Now(), datetime)

	// Format time
	hourStr := fmt.Sprintf("%02d", datetime.Hour)
	minuteStr := fmt.Sprintf("%02d", datetime.Minute)
	text := fmt.Sprintf("%s を %s:%s から！", gameName, hourStr, minuteStr)

	// Generate message components
	embed, components := generateVoteMessage(gameName, nil, nil, nil)

	// Send response
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    text,
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
		return
	}

	// Get the message we just sent
	msg, err := s.InteractionResponse(i.Interaction)
	if err != nil {
		log.Printf("Error getting interaction response: %v", err)
		return
	}

	voteID, err := CreateGameVoteMaster(gameName, targetTime, msg.ID)
	if err != nil {
		log.Printf("DB error: %v", err)
		return
	}

	// Start notifier in goroutine
	go sendMessageAt(s, i.ChannelID, voteID, gameName, targetTime)
}

func handleButtonInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Acknowledge the button press
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})

	messageID := i.Message.ID
	userID := i.Member.User.ID
	buttonID := i.MessageComponentData().CustomID

	voteMaster, err := GetVoteMasterByMessageID(messageID)
	if err != nil {
		log.Printf("failed to get vote master")
		return
	}

	var voteType VoteType
	switch buttonID {
	case VoteTypeIDYes:
		voteType = VoteTypeYes
	case VoteTypeIDNo:
		voteType = VoteTypeNo
	case VoteTypeIDPending:
		voteType = VoteTypePending
	}

	if err := VoteToGame(voteMaster.ID, userID, voteType); err != nil {
		log.Printf("failed to vote: %v", err)
		return
	}

	// Update the message
	updateInteraction(s, i, voteMaster.ID)
}

func updateInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, voteMasterID uint) {
	voteResult, err := GetVoteResult(voteMasterID)
	if err != nil {
		log.Printf("failed to get vote result: %v", err)
		return
	}

	// Get game name from embed title
	gameName := i.Message.Embeds[0].Title

	// Generate updated message components
	embed, components := generateVoteMessage(gameName, voteResult.Yes, voteResult.No, voteResult.Pending)

	// Edit the message
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content:    &i.Message.Content,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})

	if err != nil {
		log.Printf("Error updating message: %v", err)
	}
}

func sendMessageAt(s *discordgo.Session, channelID string, voteMasterID uint, gameName string, targetTime time.Time) {
	log.Printf("Notifier received: %s, %d, %s, %v", channelID, voteMasterID, gameName, targetTime)

	// Create notifier for the specified time
	notifier := gamebot.NewNotifier(targetTime, func() {
		// This function is called when the time is reached

		yesUsers, err := getYesOrPendingUsers(voteMasterID)
		if err != nil {
			log.Printf("failed to get yes or pending users")
			return
		}

		if len(yesUsers) == 0 {
			log.Printf("No users voted yes or pending")
			return
		}

		// Create mentions string
		var userMentions []string
		for _, userID := range yesUsers {
			userMentions = append(userMentions, fmt.Sprintf("<@%s>", userID))
		}
		userStr := strings.Join(userMentions, " ")

		// Create and send the message
		content := fmt.Sprintf("%s %s の時間です！", userStr, gameName)
		log.Printf("Notifier sending message: %s", content)

		_, err = s.ChannelMessageSend(channelID, content)
		if err != nil {
			log.Printf("Error sending notification message: %v", err)
		}
	})

	// Start the notifier
	notifier.WaitAndDo(context.Background())
}
