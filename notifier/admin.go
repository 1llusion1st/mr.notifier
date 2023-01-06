package notifier

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"sync"
)

func NewAdminHandler(configPath string, bot *tgbotapi.BotAPI, config *Config) *AdminHandler {
	config.setSyncPath(configPath)
	return &AdminHandler{
		Config:     config,
		Bot:        bot,
		ConfigPath: configPath,
		callbacks:  make(map[string]Command),
	}
}

type AdminHandler struct {
	Bot        *tgbotapi.BotAPI
	ConfigPath string
	Config     *Config
	callbacks  map[string]Command
	lock       sync.Mutex
}

func (a *AdminHandler) HandleUpdates(updatesChan tgbotapi.UpdatesChannel) {
	for update := range updatesChan {

		var err error = nil

		if update.CallbackQuery != nil {
			logrus.Debugf("CALLBACK [%d] %s (chat: %d): %s",
				update.CallbackQuery.From.ID, update.CallbackQuery.From.UserName,
				update.CallbackQuery.Message.MessageID, update.CallbackQuery.Data)
			// handle callback
			if a.Config.UnexpectedChat(int64(update.CallbackQuery.From.ID)) {
				continue
			}
			callback, ok := a.callbacks[update.CallbackQuery.Data]
			if !ok {
				logrus.Debugf("callback not found!")
				continue
			}
			logrus.Debugf("callback: %v", callback)
			err = callback.Execute(a.Config, a.Bot, a, update.CallbackQuery.Message.MessageID)
			if err != nil {
				logrus.Errorf("callback call(%v) error: %v", callback, err)
			} else {
				logrus.Debugf("callback executed")
			}
		} else if update.Message != nil {
			logrus.Printf("MESSAGE [%s] %s (chat: %d)", update.Message.From.UserName, update.Message.Text, update.Message.Chat.ID)
			if a.Config.UnexpectedChat(update.Message.Chat.ID) {
				continue
			}
			// handle message or command
			switch update.Message.Text {
			case "/start":
				{
					err = (&CommandStart{}).Execute(a.Config, a.Bot, a, 0)
					break
				}
			case "Projects":
				{
					err = (&CommandListProjects{}).Execute(a.Config, a.Bot, a, 0)
					break
				}
			case "Reviewers":
				{
					err = (&CommandListReviewers{}).Execute(a.Config, a.Bot, a, 0)
					break
				}
			default:

			}
		}
		if err != nil {
			logrus.Errorf("%s execution failed: %v", update.Message.Text, err)
		}
	}
}

func (a *AdminHandler) NewCallbackButton(text string, command Command) tgbotapi.InlineKeyboardButton {
	callbackId := uuid.New()
	a.lock.Lock()
	defer a.lock.Unlock()
	a.callbacks[callbackId.String()] = command

	return tgbotapi.NewInlineKeyboardButtonData(text, callbackId.String())
}
