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
			Style:    discordgo.SecondaryButton,
			CustomID: "origin",
		},
		{
			Label:    "Translate",
			Style:    discordgo.SecondaryButton,
			CustomID: "translate",
		},
		{
			Label:    "Spoiler",
			Style:    discordgo.SecondaryButton,
			CustomID: "spoiler",
		},
		{
			Label:    "Delete",
			Style:    discordgo.SecondaryButton,
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

// m (MessageCreate) 依存を排除し、REST API・Discordイベント双方から呼び出せるように変更
func SendCovertedMessage(s *discordgo.Session, channelID string, authorName string, originalContent string, convertedContent string) {
	re := regexp.MustCompile(`https?://[^\s<>]+`)
	originalURL := re.FindString(originalContent)
	if originalURL == "" {
		originalURL = originalContent
	}

	convertedContent, _, _ = strings.Cut(convertedContent, "?")

	if strings.HasPrefix(convertedContent, "https://fxtwitter.com") {
		_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Content:    "User: " + authorName + "\n" + convertedContent + "/ja",
			Components: createButtons(originalURL, []string{"Open", "Original", "Spoiler", "Delete"}),
		})
		if err != nil {
			log.Printf("メッセージ送信失敗: %v", err)
		}
	} else {
		_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Content:    "`replaced message sent by: " + authorName + "`\n" + convertedContent + "/ja",
			Components: createButtons(originalURL, []string{"Open", "Spoiler", "Delete"}),
		})
		if err != nil {
			log.Printf("メッセージ送信失敗: %v", err)
		}
	}
}

func LinkFixer(s *discordgo.Session, m *discordgo.MessageCreate, url *string, channelID *string, userID *string) {
	// REST API経由で送られてきた場合の処理
	if url != nil && channelID != nil && *url != "" && *channelID != "" {
		targetURL := *url
		targetChannel := *channelID
		author := "API"
		if userID != nil && *userID != "" {
			user, err := s.User(*userID)
			if err == nil {
				author = user.DisplayName()
			}
		}

		for _, rule := range conversionRules {
			if !slices.Contains(rule.exceptions, targetURL) {
				if strings.Contains(targetURL, rule.prefix) {
					converted := strings.ReplaceAll(targetURL, rule.prefix, rule.replaceTo)
					SendCovertedMessage(s, targetChannel, author, targetURL, converted)
					return
				}
			}
		}
		return
	}

	// 通常のDiscordメッセージイベント処理
	if m == nil || m.Author == nil || m.Author.ID == s.State.User.ID || m.GuildID == storage.Envs.IGNORE_GUILD_ID {
		return
	}

	for _, rule := range conversionRules {
		if !slices.Contains(rule.exceptions, m.Content) {
			if strings.Contains(m.Content, rule.prefix) {
				converted := strings.ReplaceAll(m.Content, rule.prefix, rule.replaceTo)
				SendCovertedMessage(s, m.ChannelID, m.Author.DisplayName(), m.Content, converted)
				s.ChannelMessageDelete(m.ChannelID, m.ID)
				break
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

		// 要素数のチェックを追加してインデックス範囲外エラーを防止
		if len(contents) > 1 && strings.Contains(contents[1], "|") {
			cleanedContent := strings.ReplaceAll(contents[1], "||", "")
			resultContent = contents[0] + "\n" + cleanedContent
		} else if len(contents) > 1 {
			resultContent = contents[0] + "\n||" + strings.Join(contents[1:], "\n") + " ||"
		} else {
			resultContent = i.Message.Content
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

		if len(url) > 1 {
			logContent := "Deleted message by: " + operator + "\nContent: " + url[1]
			_, err = s.ChannelMessageSend(storage.Envs.LOG_CHANNEL_ID, logContent)
			if err != nil {
				log.Printf("failed to send delete log to log channel: %v", err)
			}
		}

	case "origin":
		content := strings.FieldsFunc(i.Message.Content, func(r rune) bool {
			return r == '\n' || r == '\r'
		})

		if len(content) < 2 {
			return
		}

		spoilered := strings.Contains(content[1], "||")
		cleanURL := strings.ReplaceAll(content[1], "||", "")
		cleanURL = strings.TrimSpace(cleanURL)
		rawURL := strings.TrimSuffix(cleanURL, "/ja")

		displayContent := rawURL
		if spoilered {
			displayContent = "||" + rawURL + " ||"
		}

		convertedContent := content[0] + "\n" + displayContent
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

		if len(content) < 2 {
			return
		}

		spoilered := strings.Contains(content[1], "||")
		cleanURL := strings.ReplaceAll(content[1], "||", "")
		cleanURL = strings.TrimSpace(cleanURL)
		rawURL := cleanURL + "/ja"
		displayContent := rawURL
		if spoilered {
			displayContent = "||" + rawURL + " ||"
		}
		convertedContent := content[0] + "\n" + displayContent
		originalURL := strings.ReplaceAll(cleanURL, "fxtwitter.com", "x.com")
		components := createButtons(originalURL, []string{"Open", "Original", "Spoiler", "Delete"})
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
