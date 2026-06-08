package discord

import (
	"log"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/oudentabetai/dc-bot/gifmaker"
	"github.com/oudentabetai/dc-bot/pterodactyl"
	"github.com/oudentabetai/dc-bot/storage"
	"github.com/oudentabetai/dc-bot/utils"
)

func HelpCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {

	embed := &discordgo.MessageEmbed{
		Title:       "ヘルプ — コマンド一覧",
		Description: "よく使うコマンドの説明と使用例です。必要に応じてオートコンプリートを使ってください。",
		Color:       0x5865f2,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "/servers",
				Value:  "サーバー一覧を表示します。通常はアクセス可能なサーバーだけが表示されますが、owner は全件表示されます。",
				Inline: false,
			},
			{
				Name:   "/server",
				Value:  "`/server server_name:<名前>` または ` /server server_identifier:<識別子> action:<start|stop|restart|info>` の形式で使用します。\n例: `/server server_name:example action:start`",
				Inline: false,
			},
			{
				Name:   "/role",
				Value:  "ロールにサーバーを紐付けます。`/role action:add role:@role server_identifier:<識別子>` のように使用します。`list` で登録済みを確認できます。",
				Inline: false,
			},
			{
				Name:   "補足",
				Value:  "このボットではサーバー名のオートコンプリートをサポートしています。すべての応答は一時表示（ephemeral）されます。",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "Panel: https://web.ofton.dev"},
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:  discordgo.MessageFlagsEphemeral,
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		return
	}
}

func ServersCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// 1. 保留応答
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Println("InteractionRespond error:", err)
		return
	}

	// 2. サーバー情報取得
	accesibleServers := utils.GetAccessibleServers(i.Member)

	if accesibleServers == nil {
		log.Println("Error creating embed from server response")
		errMsg := "❌ サーバー情報の整形中にエラーが発生しました。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &errMsg,
		})
		return
	}
	// 3. 正常終了（Embedを表示）
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{utils.ListServers(accesibleServers)},
	})

	if err != nil {
		log.Println("Embed送信エラー:", err)
	}
}

func ServerCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Println("InteractionRespond error:", err)
		return
	}

	var action, identifier string
	for _, option := range i.ApplicationCommandData().Options {
		if option.Name == "action" {
			action = option.StringValue()
		}
		if option.Name == "server_name" {
			identifier = option.StringValue()
		}
		if option.Name == "server_identifier" {
			identifier = option.StringValue()
		}
	}
	if (action == "" && identifier == "") || (action != "" && identifier == "") {
		errorMsg := "❌ コマンドの引数が不足しています。例: !!server <action> <serverIdentifier>"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &errorMsg,
		})
		return
	}
	if action == "" {
		servers := utils.GetAccessibleServers(i.Member)
		for _, server := range servers {
			if server.Attributes.Identifier == identifier {
				embed := utils.ServerDetailEmbed(server)
				_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{embed},
				})
				return
			}
		}
		errorMsg := "❌ 指定されたサーバーが見つかりません。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &errorMsg,
		})
	} else {
		if action == "info" {
			servers := utils.GetAccessibleServers(i.Member)
			for _, server := range servers {
				if server.Attributes.Identifier == identifier {
					embed := utils.ServerDetailEmbed(server)
					_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Embeds: &[]*discordgo.MessageEmbed{embed},
					})
					return
				}
			}
		}
		servers := utils.GetAccessibleServers(i.Member)
		var Result string
		for _, server := range servers {
			if server.Attributes.Identifier != identifier {
				continue
			} else {
				Result = pterodactyl.PowerServer(identifier, action)
				_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: &Result,
				})
				break
			}
		}
		if Result == "" {
			errorMsg := "❌ 指定されたサーバーが見つかりません。"
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &errorMsg,
			})
		}
	}
}

func RoleCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Println("InteractionRespond error:", err)
		return
	}

	if i.Member.User.ID != OWNER_ID {
		errorMsg := "❌ このコマンドを使用する権限がありません。"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &errorMsg,
		})
		return
	}

	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	var (
		roleID     string
		identifier string
		action     string
	)

	if opt, ok := optionMap["role"]; ok {
		if r := opt.RoleValue(s, i.GuildID); r != nil {
			roleID = r.ID
			log.Printf("選択されたロールID: %s, 名前: %s", r.ID, r.Name)
		}
	}

	if opt, ok := optionMap["server_identifier"]; ok {
		identifier = opt.StringValue()
		log.Printf("選択されたサーバー識別子: %s", identifier)
	}

	if opt, ok := optionMap["action"]; ok {
		action = opt.StringValue()
		log.Printf("選択されたアクション: %s", action)
	}

	if (action != "" && identifier == "" && roleID != "") || (action == "list") {
		results := storage.ConfigMgr.GetRole(roleID)
		resultText := strings.Join(results, "\n")

		if resultText == "" {
			resultText = "該当するデータがありませんでした。"
		}

		_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &resultText,
		})
		if err != nil {
			log.Printf("レスポンス編集失敗: %v", err)
		}
	}
	if action == "add" {
		result := storage.ConfigMgr.SetRole(roleID, identifier)
		_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &result,
		})
		if err != nil {
			log.Printf("レスポンス編集失敗: %v", err)
		}
	}
	if action == "remove" {
		result := storage.ConfigMgr.RemoveRole(roleID, identifier)
		_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &result,
		})
		if err != nil {
			log.Printf("レスポンス編集失敗: %v", err)
		}
	}
}

func GifCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// 1. 即座に「考え中...」のレスポンスを返す（3秒ルール対策）
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Println("InteractionRespond error:", err)
		return
	}

	// 2. オプションの存在チェックと型チェック
	options := i.ApplicationCommandData().Options
	if len(options) == 0 || options[0].Type != discordgo.ApplicationCommandOptionAttachment {
		errorMsg := "❌ 添付ファイルが必要です。"
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &errorMsg,
		})
		return
	}

	// 3. 添付ファイルのURLを取得（安全のためProxyURLを推奨）
	attachmentID := options[0].Value.(string)
	attachment := i.ApplicationCommandData().Resolved.Attachments[attachmentID]
	attachmentURL := attachment.ProxyURL // URL でも動きますが、ProxyURL の方が確実です

	// 4. GIF変換処理の実行
	file, err := gifmaker.ConvertToGif(attachmentURL)
	if err != nil {
		log.Printf("GIF変換エラー: %v", err)
		errorMsg := "❌ GIFの生成に失敗しました。"
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &errorMsg,
		})
		return
	}
	// 関数が正常終了、またはエラー終了する際に、生成された一時GIFファイルを必ず削除する
	defer os.Remove("out.gif")

	msg, err := s.ChannelMessageSendComplex(storage.Envs.LOG_CHANNEL_ID, &discordgo.MessageSend{
		Files: []*discordgo.File{
			{
				Name:   "animation.gif",
				Reader: file,
			},
		},
	})
	if err != nil {
		log.Println("メッセージ送信エラー:", err)
		return
	}
	attachmentURL = msg.Attachments[0].URL

	// 6. 完了したGIFファイルをDiscordに送信（InteractionResponseEdit）
	successMsg := "🎉 GIFの生成が完了しました！\n" + attachmentURL
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &successMsg,
	})
	if err != nil {
		log.Printf("レスポンス編集失敗: %v", err)
	}
}
