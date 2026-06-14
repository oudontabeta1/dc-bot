package discord

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/oudentabetai/dc-bot/gifmaker"
	"github.com/oudentabetai/dc-bot/linkfixer"
	"github.com/oudentabetai/dc-bot/pterodactyl"
	"github.com/oudentabetai/dc-bot/storage"
)

var (
	suffix string = "!!"
)

type SessionManager interface {
	InitializeSession(token string) *discordgo.Session
}

type DiscordSessionManager struct{}

func (d *DiscordSessionManager) InitializeSession(token string) *discordgo.Session {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Discordセッションの作成に失敗: %v", err)
	}
	return dg
}

func OnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic recovered in OnMessageCreate: %v", r)
		}
	}()

	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content == "!gif" {
		var att *discordgo.MessageAttachment
		if len(m.Attachments) > 0 {
			att = m.Attachments[0]
		} else if m.ReferencedMessage != nil {
			att = m.ReferencedMessage.Attachments[0]
		}
		switch filepath.Ext(att.Filename) {
		case ".png", ".jpg", ".jpeg", ".webp", ".heic", ".heif":
			log.Print(att.URL)
			file, err := gifmaker.ConvertToGif(att.URL)
			if err != nil {
				log.Printf("Failed to Convert gif: %v", err)
				return
			}

			message := &discordgo.MessageSend{
				Files: []*discordgo.File{
					{
						Name:   "out.gif",
						Reader: file,
					},
				},
				Reference: &discordgo.MessageReference{
					MessageID: m.ID,
					ChannelID: m.ChannelID,
					GuildID:   m.GuildID,
				},
			}

			_, err = s.ChannelMessageSendComplex(m.ChannelID, message)
			if err != nil {
				log.Printf("メッセージ送信エラー: %v", err)
			}
		}
	} else {
		linkfixer.LinkFixer(s, m)
	}
}

//TextCommand(s, m)

func TextCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	u := m.Author
	if !u.Bot {
		if strings.HasPrefix(m.Content, suffix) {
			log.Printf("Received command: %s from user: %s", m.Content, u.Username)
		}
		if strings.HasPrefix(m.Content, suffix+"user") {
			s.ChannelMessageSend(m.ChannelID, pterodactyl.GetUser())
		}
		if strings.HasPrefix(m.Content, suffix+"servers") {

		}
		if strings.HasPrefix(m.Content, suffix+"setrole") {
			ownerID := storage.Envs.OWNER_ID
			if m.Author.ID != ownerID {
				log.Print("User does not have permission to set role: " + ownerID + " vs " + m.Author.ID)
				s.ChannelMessageSend(m.ChannelID, "このコマンドを使用する権限がありません。")
				return
			}
			fields := strings.Fields(m.Content)
			if len(fields) == 3 {
				s.ChannelMessageSend(m.ChannelID, storage.ConfigMgr.SetRole(fields[1], fields[2]))
			} else {
				s.ChannelMessageSend(m.ChannelID, "コマンドの形式が正しくありません。例: !!setrole <roleID> <serverID>")
			}
		}
		if strings.HasPrefix(m.Content, suffix+"server") {
			fields := strings.Fields(m.Content)
			if len(fields) == 3 {
				s.ChannelMessageSend(m.ChannelID, pterodactyl.PowerServer(fields[1], fields[2]))
			} else {
				s.ChannelMessageSend(m.ChannelID, "コマンドの形式が正しくありません。例: !!server <action> <serverIdentifier>")
			}
		}
		if strings.HasPrefix(m.Content, suffix+"test") {
		}
	}
}
