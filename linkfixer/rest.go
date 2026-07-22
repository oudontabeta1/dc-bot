package linkfixer

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/gin-gonic/gin"
)

type Data struct {
	URL       *string `json:"url"`
	UserID    *string `json:"user_id"`
	ChannelID *string `json:"channel_id"`
}

type Server struct {
	Session *discordgo.Session
}

func NewServer(s *discordgo.Session) *Server {
	return &Server{
		Session: s,
	}
}

func (s *Server) sendLinkFixer(c *gin.Context) {
	var data Data
	if err := c.BindJSON(&data); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	log.Printf("Received API data: %+v", data)

	// 受け取ったデータを LinkFixer へ渡し、URLの置換・メッセージ送信・ボタン追加を実行します
	LinkFixer(s.Session, nil, data.URL, data.ChannelID, data.UserID)

	c.JSON(200, gin.H{"status": "ok"})
}

func (s *Server) Start(addr string) {
	router := gin.Default()
	router.POST("/linkfixer", s.sendLinkFixer)

	// 別ゴルーチンでGinサーバーを起動
	go func() {
		if err := router.Run(addr); err != nil {
			log.Fatalf("Failed to run HTTP server: %v", err)
		}
	}()
}
