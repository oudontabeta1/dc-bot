package discord

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/oudentabetai/dc-bot/linkfixer"
	"github.com/oudentabetai/dc-bot/storage"
	"github.com/oudentabetai/dc-bot/utils"
)

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name: "help",
			NameLocalizations: &map[discordgo.Locale]string{
				discordgo.Locale("ja"):    "ヘルプ",
				discordgo.Locale("en-US"): "help",
			},
			Description: "Help Command",
			DescriptionLocalizations: &map[discordgo.Locale]string{
				discordgo.Locale("ja"):    "ヘルプコマンド",
				discordgo.Locale("en-US"): "Help Command",
			},
		},
		{
			Name: "servers",
			NameLocalizations: &map[discordgo.Locale]string{
				discordgo.Locale("ja"):    "サーバー一覧",
				discordgo.Locale("en-US"): "servers",
			},
			Description: "List serverlist",
			DescriptionLocalizations: &map[discordgo.Locale]string{
				discordgo.Locale("ja"):    "サーバー一覧表示",
				discordgo.Locale("en-US"): "List serverlist",
			},
		},
		{
			Name: "server",
			NameLocalizations: &map[discordgo.Locale]string{
				discordgo.Locale("ja"):    "サーバー",
				discordgo.Locale("en-US"): "server",
			},
			Description: "Manage Server",
			DescriptionLocalizations: &map[discordgo.Locale]string{
				discordgo.Locale("ja"):    "サーバーの管理",
				discordgo.Locale("en-US"): "Manage Server",
			},
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "server_id",
					Description: "ServerId(Identifier)",
					Required:    false,
					Type:        discordgo.ApplicationCommandOptionString,
				},
				{
					Name:         "server_name",
					Description:  "Server Name",
					Required:     false,
					Autocomplete: true,
					Type:         discordgo.ApplicationCommandOptionString,
				},
				{
					Name:        "action",
					Description: "Action that you want",
					Required:    false,
					Type:        discordgo.ApplicationCommandOptionString,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "Start",
							Value: "start",
						},
						{
							Name:  "Stop",
							Value: "stop",
						},
						{
							Name:  "Restart",
							Value: "restart",
						},
						{
							Name:  "Information",
							Value: "info",
						},
					},
				},
			},
		},
		{
			Name: "role",
			NameLocalizations: &map[discordgo.Locale]string{
				discordgo.Locale("ja"):    "ロール",
				discordgo.Locale("en-US"): "role",
			},
			Description: "Manage role",
			DescriptionLocalizations: &map[discordgo.Locale]string{
				discordgo.Locale("ja"):    "ロール管理",
				discordgo.Locale("en-US"): "Manage role",
			},
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "action",
					Description: "Action that you want",
					Required:    true,
					Type:        discordgo.ApplicationCommandOptionString,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "List",
							Value: "list",
						},
						{
							Name:  "Add",
							Value: "add",
						},
						{
							Name:  "Remove",
							Value: "remove",
						},
					},
				},
				{
					Name:        "role",
					Description: "Role that you want to manage",
					Required:    false,
					Type:        discordgo.ApplicationCommandOptionRole,
				},
				{
					Name:        "server_identifier",
					Description: "Server Identifier",
					Required:    false,
					Type:        discordgo.ApplicationCommandOptionString,
				},
			},
		},
		{
			Name: "gif",
			NameLocalizations: &map[discordgo.Locale]string{
				discordgo.Locale("ja"):    "gif",
				discordgo.Locale("en-US"): "gif",
			},
			Description: "Create a gif from images",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "image",
					Description: "Image to include in the gif (up to 10)",
					Required:    true,
					Type:        discordgo.ApplicationCommandOptionAttachment,
				},
			},
		},
	}
	CommandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"help":    HelpCommandHandler,
		"servers": ServersCommandHandler,
		"server":  ServerCommandHandler,
		"role":    RoleCommandHandler,
		"gif":     GifCommandHandler,
	}
)

func SyncCommands(s *discordgo.Session, guildID string, appID string) {
	_, err := s.ApplicationCommandBulkOverwrite(appID, guildID, commands)
	if err != nil {
		log.Panicf("コマンドの同期に失敗しました: %v", err)
	}
	log.Println("コマンドを更新しました")
}

func OnInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionMessageComponent {
		linkfixer.OnButton(s, i)
		return
	}
	if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
		respondAutocomplete(s, i)
		return
	}

	if i.Type == discordgo.InteractionApplicationCommand {
		name := i.ApplicationCommandData().Name
		handler, ok := CommandHandlers[name]
		if !ok {
			return
		}

		// 💡 ログ用にオプションを安全に文字列化する
		var options []string
		for _, opt := range i.ApplicationCommandData().Options {
			var valStr string

			// 型に応じて安全に文字列に変換
			switch opt.Type {
			case discordgo.ApplicationCommandOptionString:
				valStr = opt.StringValue()
			case discordgo.ApplicationCommandOptionRole:
				// ロール型の場合はIDを文字列にする（あるいは Name が取れれば Name でも良い）
				if r := opt.RoleValue(s, i.GuildID); r != nil {
					valStr = fmt.Sprintf("%s(%s)", r.Name, r.ID)
				} else {
					valStr = fmt.Sprintf("%v", opt.Value)
				}
			default:
				// その他の型（Integer, Boolean, Userなど）はGoの標準フォーマットで文字列化
				valStr = fmt.Sprintf("%v", opt.Value)
			}

			options = append(options, fmt.Sprintf("%s: %s", opt.Name, valStr))
		}

		// 先にハンドラを実行
		handler(s, i)

		if storage.Envs.LOG_CHANNEL_ID != "" {
			go func(commandName, username string, commandOptions []string) {
				// ログのフォーマットを少し見やすく調整
				msg := fmt.Sprintf("【コマンドログ】\n実行者: %s\nコマンド: /%s\nオプション: %s",
					username, commandName, strings.Join(commandOptions, ", "))

				_, err := s.ChannelMessageSend(storage.Envs.LOG_CHANNEL_ID, msg)
				if err != nil {
					log.Printf("ログチャンネルへの送信に失敗: %v", err)
				}
			}(name, i.Member.User.Username, options)
		}
	}
}

func respondAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, 25)

	for _, opt := range data.Options {
		if !opt.Focused {
			continue
		}

		userInput := strings.ToLower(opt.StringValue())
		servers := utils.GetAccessibleServers(i.Member)
		for _, srv := range servers {
			name := srv.Attributes.Name
			if userInput == "" || strings.Contains(strings.ToLower(name), userInput) {
				choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
					Name:  name,
					Value: srv.Attributes.Identifier,
				})
				if len(choices) >= 25 {
					break
				}
			}
		}
		break
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseType(8),
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
	if err != nil {
		log.Printf("autocomplete respond error: %v", err)
	}
}
