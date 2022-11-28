package main

import (
	"encoding/json"
	"fmt"
	"github.com/mymmrac/telego"
	. "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"io/ioutil"
	"log"
	"os"
)

type Settings struct {
	ChannelID    int64 `json:"channelID"`
	SupergroupID int64 `json:"supergroupID"`
}

var settings Settings
var botUser *telego.User

func main() {
	file, _ := ioutil.ReadFile("Settings.json")
	settings = Settings{}
	_ = json.Unmarshal(file, &settings)

	botToken := os.Getenv("TOKEN")

	// Note: Please keep in mind that default logger may expose sensitive information,
	// use in development only
	bot, err := telego.NewBot(botToken, telego.WithDefaultDebugLogger())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// check and delete webhook
	wh, _ := bot.GetWebhookInfo()
	if wh.URL != "" {
		log.Println("Webhook status: ", wh)
		if err := bot.DeleteWebhook(nil); err != nil {
			log.Println("deleteWebhook ERR: ", err)
		} else {
			log.Println("deleteWebhook SUCCESSFUL: ")
		}
	}

	SetBotCommands(bot)

	// Call method getMe
	botUser, _ = bot.GetMe()
	fmt.Printf("Bot User: %+v\n", botUser)

	allowedUpdates := []string{"message", "callback_query"}
	params := telego.GetUpdatesParams{
		Offset:         0,
		Limit:          100,
		Timeout:        10,
		AllowedUpdates: allowedUpdates,
	}

	updates, _ := bot.UpdatesViaLongPulling(&params)

	// Create bot handler and specify from where to get updates
	bh, _ := NewBotHandler(bot, updates)

	// Stop handling updates
	defer bh.Stop()

	// Stop getting updates
	defer bot.StopLongPulling()

	// Handler channel posts
	bh.HandleChannelPost(func(bot *telego.Bot, message telego.Message) {
		//TODO: make saving channel post to DB

	}, AnyChannelPost())

	//A handlers functions contain in handlers.go
	//Handle a private chat commands
	bh.HandleMessage(handlePrivateCommands, AnyCommand(),
		func(update telego.Update) bool {
			return update.Message.Chat.Type == "private"
		})

	//Handle a messages from user
	bh.HandleMessage(handlePrivateMessage,
		Not(AnyCommand()),
		func(update telego.Update) bool {
			return update.Message.Chat.Type == "private"
		})

	// Handle a supergroup(discussion) messages forwarded form channel
	bh.HandleMessage(handleGroupPostFromChannel, func(update telego.Update) bool {
		return update.Message.From.ID == 777000 &&
			update.Message.IsAutomaticForward &&
			update.Message.Chat.ID == settings.SupergroupID &&
			update.Message.SenderChat.ID == settings.ChannelID
	})

	// Handle reply in supergroup
	bh.HandleMessage(handleReplyGroupMessage, func(update telego.Update) bool {
		if update.Message.Chat.ID == settings.SupergroupID &&
			update.Message.ReplyToMessage != nil {
			return update.Message.ReplyToMessage.From.ID == botUser.ID
		}
		return false
	})

	// handle CLOSE command
	bh.HandleMessage(handleGroupCommands, AnyCommand(), func(update telego.Update) bool {
		return update.Message.Chat.ID == settings.SupergroupID
	})

	//Handle SuperGroup Callback
	bh.HandleCallbackQuery(handleCallbackQuery, AnyCallbackQueryWithMessage(), func(update telego.Update) bool {
		return update.CallbackQuery.Message.Chat.ID == settings.SupergroupID
	})

	// Start handling updates
	bh.Start()
}

func deleteInlineKeyboard(messageID int, chatID int64) *telego.EditMessageReplyMarkupParams {
	editMarkupParams := telego.EditMessageReplyMarkupParams{}
	emptyKeyboard := editMarkupParams.WithMessageID(messageID).
		WithChatID(tu.ID(chatID)).WithReplyMarkup(
		&telego.InlineKeyboardMarkup{
			InlineKeyboard: make([][]telego.InlineKeyboardButton, 0),
		})
	return emptyKeyboard
}
