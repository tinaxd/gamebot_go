package main

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const (
	VoteTypeIDYes     = "game_yes"
	VoteTypeIDNo      = "game_no"
	VoteTypeIDPending = "game_pending"
)

var (
	appCommand = &discordgo.ApplicationCommand{
		Name:        "game",
		Description: "ゲームの予定をたてる",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "game",
				Description: "プレイするゲーム",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "time",
				Description: "ゲームの開始時間(4ケタ)。指定しなくてもよい。",
				Required:    false,
			},
		},
	}
)

func generateVoteMessage(gameName string, yesUsers, noUsers, pendingUsers []string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	// Create action row with buttons
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "やる",
					CustomID: VoteTypeIDYes,
					Style:    discordgo.PrimaryButton,
					Emoji: &discordgo.ComponentEmoji{
						Name: "😆",
					},
				},
				discordgo.Button{
					Label:    "できない",
					CustomID: VoteTypeIDNo,
					Style:    discordgo.SecondaryButton,
					Emoji: &discordgo.ComponentEmoji{
						Name: "😭",
					},
				},
				discordgo.Button{
					Label:    "未定",
					CustomID: VoteTypeIDPending,
					Style:    discordgo.SecondaryButton,
					Emoji: &discordgo.ComponentEmoji{
						Name: "🤔",
					},
				},
			},
		},
	}

	// Create embed
	embed := &discordgo.MessageEmbed{
		Title:       gameName,
		Description: fmt.Sprintf("%s やる人集まれ～", gameName),
		Color:       0xF1C40F, // Yellow color
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "やる😆",
				Value:  generateUserLine(yesUsers),
				Inline: true,
			},
			{
				Name:   "できない😭",
				Value:  generateUserLine(noUsers),
				Inline: true,
			},
			{
				Name:   "未定🤔",
				Value:  generateUserLine(pendingUsers),
				Inline: true,
			},
		},
	}

	return embed, components
}

func generateUserLine(users []string) string {
	userLineDivider := "--------------------"
	userCount := len(users)

	var userStr string
	if len(users) > 0 {
		var userMentions []string
		for _, userID := range users {
			userMentions = append(userMentions, fmt.Sprintf("<@%s>", userID))
		}
		userStr = strings.Join(userMentions, "\n")
	}

	return fmt.Sprintf("%d\n%s\n%s", userCount, userLineDivider, userStr)
}
