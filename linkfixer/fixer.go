package linkfixer

import (
	"log"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/oudentabetai/dc-bot/storage"
)

// 各サービスのプレフィックスと変換先のマップ
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
		exceptions: []string{}, // 元のコードの exceptions に合わせる場合ここで指定
	},
}

// ConvertMessage はメッセージ内のURLを条件に応じて変換する
func ConvertMessage(msg string) (string, bool) {
	for _, rule := range conversionRules {
		// 例外条件に完全に一致する場合は処理をスキップ
		isException := false
		for _, ex := range rule.exceptions {
			if msg == ex {
				isException = true
				break
			}
		}
		if isException {
			continue
		}

		// URLが含まれているかチェック
		if strings.Contains(msg, rule.prefix) {
			converted := strings.ReplaceAll(msg, rule.prefix, rule.replaceTo)
			return converted, true
		}
	}

	return "", false // 変換が行われなかった場合
}

func firstURL(text string) string {
	re := regexp.MustCompile(`https?://[^\s<>]+`)
	return re.FindString(text)
}

func SendCovertedMessage(s *discordgo.Session, m *discordgo.MessageCreate, originalContent string, convertedContent string) {
	originalURL := firstURL(originalContent)
	if originalURL == "" {
		originalURL = originalContent
	}

	_, err := s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Content: "`" + "replaced message sent by: " + m.Author.Username + "`" + "\n" + convertedContent,
		Components: []discordgo.MessageComponent{
			&discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					// 1つ目のボタン
					discordgo.Button{
						Label: "Open",
						Style: discordgo.LinkButton,
						URL:   originalURL,
					},
					// 2つ目のボタン
					discordgo.Button{
						Label:    "Spoiler",
						Style:    discordgo.PrimaryButton,
						CustomID: "spoiler",
					},
					// 3つ目のボタン
					discordgo.Button{
						Label:    "Delete",
						Style:    discordgo.DangerButton,
						CustomID: "delete",
					},
				},
			},
		},
	})

	if err != nil {
		log.Printf("メッセージ送信失敗: %v", err)
	}
}

func sendDeleteLog(s *discordgo.Session, fallbackChannelID string, content string) {
	logChannelID := storage.Envs.LOG_CHANNEL_ID
	if logChannelID != "" {
		if _, err := s.ChannelMessageSend(logChannelID, content); err == nil {
			return
		} else {
			log.Printf("failed to send delete log to log channel: %v", err)
		}
	}

	if fallbackChannelID != "" {
		if _, err := s.ChannelMessageSend(fallbackChannelID, content); err != nil {
			log.Printf("failed to send delete log to fallback channel: %v", err)
		}
	}
}

func Main(s *discordgo.Session, m *discordgo.MessageCreate) {
	// メッセージがボット自身のものであれば無視
	if m.Author.ID == s.State.User.ID {
		return
	}
	if m.GuildID == "1238890574132809798" {
		return
	}
	content := m.Content
	converted, changed := ConvertMessage(content)
	if changed {
		s.ChannelMessageDelete(m.ChannelID, m.ID)
		SendCovertedMessage(s, m, content, converted)
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
		// Defer the interaction first
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredMessageUpdate,
		})
		if err != nil {
			log.Printf("spoiler interaction defer failed: %v", err)
			return
		}

		var resultContent string
		contents := strings.Split(i.Message.Content, "\n")
		if strings.Contains(contents[1], "|") {
			// Remove spoiler markers
			cleanedContent := strings.ReplaceAll(contents[1], "|", "")
			resultContent = contents[0] + "\n" + cleanedContent
		} else {
			// Add spoiler markers
			resultContent = contents[0] + "\n||" + strings.Join(contents[1:], "\n") + "||"
		}

		// Edit the message after deferring
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content:    &resultContent,
			Components: &i.Message.Components,
		})
		if err != nil {
			log.Printf("spoiler interaction edit failed: %v", err)
			return
		}

	case "delete":
		// Defer the interaction first
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			log.Printf("delete interaction defer failed: %v", err)
			return
		}

		sendDeleteLog(s, i.ChannelID, "Deleted message by: "+operator+"\nContent: "+i.Message.Content)

		deleteLog := "Deleted message by: " + operator + "\n"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &deleteLog,
		})

		err = s.ChannelMessageDelete(i.ChannelID, i.Message.ID)
		if err != nil {
			log.Printf("failed to delete message: %v", err)
		}

	default:
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
		})
		if err != nil {
			log.Printf("unknown interaction ack failed: %v", err)
		}
	}
}
