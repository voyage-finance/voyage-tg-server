package service

import (
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/voyage-finance/voyage-tg-server/models"
)

type PendingQueueTask struct {
	ChatId int64
	Text   string
}

type Notification struct {
	Tasks []PendingQueueTask
	Bot   *tgbotapi.BotAPI
	S     *Service
}

func (n *Notification) Start() {
	log.Println("Notification service start...")

	for {
		log.Println("start handle notification tasks")
		log.Println("find all chats")
		chats := n.FindAllChatIds()
		for _, chat := range chats {
			log.Println("chat id: ", chat.ChatId)
		}

		for _, t := range n.Tasks {
			msg := tgbotapi.NewMessage(t.ChatId, t.Text)
			n.Bot.Send(msg)
		}
		n.Tasks = []PendingQueueTask{}

		time.Sleep(2 * time.Minute)
	}

}

func (n *Notification) AddNewPendingQueueTask(chatId int64, msg string) {
	var newTask PendingQueueTask
	newTask.ChatId = chatId
	newTask.Text = msg
	n.Tasks = append(n.Tasks, newTask)
}

func (n *Notification) FindAllChatIds() []models.Chat {
	var chats []models.Chat
	n.S.DB.Find(&chats)
	return chats
}
