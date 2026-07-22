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

type ImgSrc struct {
	URL      string
	Filename string
}

func (d *DiscordSessionManager) InitializeSession(token string) *discordgo.Session {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Discordセッションの作成に失敗: %v", err)
	}
	return dg
}

// GetImageFromMessage は添付ファイル、またはリプライ元のEmbed画像/添付ファイルから
// 画像情報を取得する。優先順位は以下の通り:
//  1. メッセージ自身の添付ファイル
//  2. リプライ元のEmbed画像
//  3. リプライ元の添付ファイル
//
// 画像が見つからない場合は空のImgSrcを返す。
func GetImageFromMessage(m *discordgo.MessageCreate) ImgSrc {
	var img ImgSrc

	// 1. 自分自身の添付ファイル
	if len(m.Attachments) > 0 {
		img.URL = m.Attachments[0].URL
		img.Filename = m.Attachments[0].Filename
		return img
	}

	// 2, 3. リプライ元から取得
	if m.ReferencedMessage != nil {
		ref := m.ReferencedMessage

		if len(ref.Embeds) > 0 && ref.Embeds[0].Image != nil && ref.Embeds[0].Image.URL != "" {
			img.URL = ref.Embeds[0].Image.URL
			img.Filename = "in.png"
			return img
		}

		if len(ref.Attachments) > 0 {
			img.URL = ref.Attachments[0].URL
			img.Filename = ref.Attachments[0].Filename
			return img
		}
	}

	return img
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
		handleGifCommand(s, m)
		return
	}

	linkfixer.LinkFixer(s, m, nil, nil, nil)
	//TextCommand(s, m)
}

func handleGifCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	image := GetImageFromMessage(m)
	if image.URL == "" {
		s.ChannelMessageSend(m.ChannelID, "画像が見つかりませんでした。添付ファイルを付けるか、画像付きメッセージにリプライしてください。")
		return
	}

	switch filepath.Ext(image.Filename) {
	case ".png", ".jpg", ".jpeg", ".webp", ".heic", ".heif":
		log.Print(image.URL)
		file, err := gifmaker.ConvertToGif(image.URL)
		if err != nil {
			log.Printf("Failed to Convert gif: %v", err)
			s.ChannelMessageSend(m.ChannelID, "GIFへの変換に失敗しました。")
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

		if _, err := s.ChannelMessageSendComplex(m.ChannelID, message); err != nil {
			log.Printf("メッセージ送信エラー: %v", err)
		}
	default:
		s.ChannelMessageSend(m.ChannelID, "対応していない画像形式です。(png, jpg, jpeg, webp, heic, heif のみ対応)")
	}
}

func TextCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	u := m.Author
	if u.Bot {
		return
	}

	if !strings.HasPrefix(m.Content, suffix) {
		return
	}

	log.Printf("Received command: %s from user: %s", m.Content, u.Username)

	switch {
	case strings.HasPrefix(m.Content, suffix+"user"):
		s.ChannelMessageSend(m.ChannelID, pterodactyl.GetUser())

	case strings.HasPrefix(m.Content, suffix+"servers"):
		// TODO: 実装予定

	case strings.HasPrefix(m.Content, suffix+"setrole"):
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

	case strings.HasPrefix(m.Content, suffix+"server"):
		fields := strings.Fields(m.Content)
		if len(fields) == 3 {
			s.ChannelMessageSend(m.ChannelID, pterodactyl.PowerServer(fields[1], fields[2]))
		} else {
			s.ChannelMessageSend(m.ChannelID, "コマンドの形式が正しくありません。例: !!server <action> <serverIdentifier>")
		}

	case strings.HasPrefix(m.Content, suffix+"test"):
		// TODO: 実装予定
	}
}
