package linkfixer

import (
	"log"
	"regexp"
	"slices"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/oudentabetai/dc-bot/storage"
)

var conversionRules = []struct {
	prefix     string
	replaceTo  string
	exceptions []string
}{
	{
		prefix:     "https://twitter.com",
		replaceTo:  "https://fxtwitter.com",
		exceptions: []string{"https://twitter.com/"},
	},
	{
		prefix:     "https://x.com",
		replaceTo:  "https://fxtwitter.com",
		exceptions: []string{"https://x.com/"},
	},
	{
		prefix:     "https://www.instagram.com",
		replaceTo:  "https://www.uuinstagram.com",
		exceptions: []string{"https://www.instagram.com/"},
	},
	{
		prefix:     "https://pixiv.net",
		replaceTo:  "https://phixiv.net",
		exceptions: []string{"https://pixiv.net/"},
	},
	{
		prefix:     "https://soundcloud.com",
		replaceTo:  "https://fxcloud.ofton.dev",
		exceptions: []string{"https://soundcloud.com/"},
	},
	{
		prefix:     "https://open.spotify.com",
		replaceTo:  "https://open.fxspotify.com",
		exceptions: []string{},
	},
}

func createButtons(originalURL string, button []string) []discordgo.MessageComponent {
	var Buttons = []discordgo.Button{
		{
			Label: "Open",
			Style: discordgo.LinkButton,
			URL:   originalURL,
		},
		{
			Label:    "Original",
			Style:    discordgo.PrimaryButton,
			CustomID: "origin",
		},
		{
			Label:    "Translate",
			Style:    discordgo.PrimaryButton,
			CustomID: "translate",
		},
		{
			Label:    "Spoiler",
			Style:    discordgo.SecondaryButton,
			CustomID: "spoiler",
		},
		{
			Label:    "Delete",
			Style:    discordgo.DangerButton,
			CustomID: "delete",
		},
	}
	var filteredButtons []discordgo.MessageComponent

	for _, btn := range Buttons {
		if slices.Contains(button, btn.Label) {
			filteredButtons = append(filteredButtons, btn)
		}
	}

	if len(filteredButtons) == 0 {
		return nil
	}

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: filteredButtons,
		},
	}
}

func SendCovertedMessage(s *discordgo.Session, m *discordgo.MessageCreate, originalContent string, convertedContent string) {
	re := regexp.MustCompile(`https?://[^\s<>]+`)
	originalURL := re.FindString(originalContent)
	if originalURL == "" {
		originalURL = originalContent
	}

	convertedContent, _, _ = strings.Cut(convertedContent, "?")

	if strings.HasPrefix(convertedContent, "https://fxtwitter.com") {
		_, err := s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Content:    "Message by: " + m.Author.Username + "\n" + convertedContent + "/ja/",
			Components: createButtons(originalURL, []string{"Open", "Original", "Spoiler", "Delete"}),
		})
		if err != nil {
			log.Printf("メッセージ送信失敗: %v", err)
		}
	} else {
		_, err := s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Content: "`" + "replaced message sent by: " + m.Author.Username + "`" + "\n" + convertedContent + "/ja",
			Components: []discordgo.MessageComponent{
				&discordgo.ActionsRow{
					Components: createButtons(originalURL, []string{"Open", "Spoiler", "Delete"}),
				},
			},
		})
		if err != nil {
			log.Printf("メッセージ送信失敗: %v", err)
		}
	}
}

func LinkFixer(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID || m.GuildID == storage.Envs.IGNORE_GUILD_ID {
		return
	}
	for _, rule := range conversionRules {
		if !slices.Contains(rule.exceptions, m.Content) {
			if strings.Contains(m.Content, rule.prefix) {
				converted := strings.ReplaceAll(m.Content, rule.prefix, rule.replaceTo)
				SendCovertedMessage(s, m, m.Content, converted)
				s.ChannelMessageDelete(m.ChannelID, m.ID)
				continue
			}
		}
	}
}

func OnButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	if i.Message == nil {
		return
	}

	operator := "unknown"
	if i.Member != nil && i.Member.User != nil {
		operator = i.Member.User.Username
	} else if i.User != nil {
		operator = i.User.Username
	}

	switch customID {
	case "spoiler":
		var resultContent string
		contents := strings.Split(i.Message.Content, "\n")
		if strings.Contains(contents[1], "|") {
			cleanedContent := strings.ReplaceAll(contents[1], "|", "")
			resultContent = contents[0] + "\n" + cleanedContent
		} else {
			resultContent = contents[0] + "\n||" + strings.Join(contents[1:], "\n") + "||"
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    resultContent,
				Components: i.Message.Components,
			},
		})

	case "delete":
		content := i.Message.Content

		err := s.ChannelMessageDelete(i.ChannelID, i.Message.ID)
		if err != nil {
			log.Printf("failed to delete message: %v", err)
		}

		deleteLog := "Message Deleted by: " + operator
		s.ChannelMessageSend(i.ChannelID, deleteLog)

		url := strings.FieldsFunc(content, func(r rune) bool {
			return r == '\n' || r == '\r'
		})

		logContent := "Deleted message by: " + operator + "\nContent: " + url[1]
		_, err = s.ChannelMessageSend(storage.Envs.LOG_CHANNEL_ID, logContent)
		if err != nil {
			log.Printf("failed to send delete log to log channel: %v", err)
		}

	case "origin":
		content := strings.FieldsFunc(i.Message.Content, func(r rune) bool {
			return r == '\n' || r == '\r'
		})
		rawURL, _, _ := strings.Cut(content[1], "/ja/")

		convertedContent := content[0] + "\n" + rawURL
		originalURL := strings.ReplaceAll(rawURL, "fxtwitter.com", "x.com")
		components := createButtons(originalURL, []string{"Open", "Translate", "Spoiler", "Delete"})

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    convertedContent,
				Components: components,
			},
		})

	case "translate":
		content := strings.FieldsFunc(i.Message.Content, func(r rune) bool {
			return r == '\n' || r == '\r'
		})
		rawURL := content[1] + "/ja/"

		convertedContent := content[0] + "\n" + rawURL

		originalURL := strings.ReplaceAll(content[1], "fxtwitter.com", "x.com")

		components := createButtons(originalURL, []string{"Open", "Original", "Spoiler", "Delete"}) // &ActionsRowで包まない

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    convertedContent,
				Components: components,
			},
		})

	default:
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
		})
		if err != nil {
			log.Printf("unknown interaction ack failed: %v", err)
		}
	}
}
