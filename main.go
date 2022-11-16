package main

import (
	"encoding/json"
	"fmt"
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"go.mongodb.org/mongo-driver/mongo"
	mdb "goland/oneWaySupportBot/mongodb"
	"gopkg.in/gookit/color.v1"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

type settings struct {
	ChannelID    int64 `json:"channelID"`
	SupergroupID int64 `json:"supergroupID"`
}

func main() {
	botToken := os.Getenv("TOKEN")

	file, _ := ioutil.ReadFile("settings.json")
	settings := settings{}
	_ = json.Unmarshal([]byte(file), &settings)

	// Note: Please keep in mind that default logger may expose sensitive information,
	// use in development only
	bot, err := telego.NewBot(botToken, telego.WithDefaultDebugLogger())

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Check and delete Webhook
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
	botUser, _ := bot.GetMe()
	fmt.Printf("Bot User: %+v\n", botUser)
	allowedUpdates := []string{"message", "channel_post", "callback_query"}
	params := telego.GetUpdatesParams{
		//Offset:         0,
		Limit:          100,
		Timeout:        10,
		AllowedUpdates: allowedUpdates,
	}

	updates, _ := bot.UpdatesViaLongPulling(&params)

	// Create bot handler and specify from where to get updates
	bh, _ := th.NewBotHandler(bot, updates)

	// Stop handling updates
	defer bh.Stop()

	// Stop getting updates
	defer bot.StopLongPulling()

	//*** PRIVATE CHATS ***
	//TODO: make forwarding private messages to channel discussion
	//TODO: make welcome message and menu
	//Handler for private chat commands
	bh.HandleMessage(func(bot *telego.Bot, message telego.Message) {
		switch message.Text {
		case "/start":
			user, ok := mdb.GetUser(message.From.ID)
			if ok != mongo.ErrNoDocuments {
				_, _ = bot.SendMessage(tu.Message(
					tu.ID(message.Chat.ID), fmt.Sprintf(
						`С возвращением, %s!
Опишите вашу проблему <b>одним сообщением</b> и мы постараемся ее решить в кратчайшин строки!`, user.FirstName),
				).WithParseMode("HTML"))
			} else {
				_, _ = bot.SendMessage(tu.Message(
					tu.ID(message.Chat.ID), fmt.Sprintf(
						`Добро пожаловать в бот технической поддержки клиентов!
Опишите вашу проблему <b>одним сообщением</b> и мы постараемся ее решить в кратчайшин строки!`),
				).WithParseMode("HTML"))
				err := mdb.AddUser(message.From)
				if err != nil {
					log.Println("error addUser: ", err)
				}
				color.Yellow.Printf("Add user to Mongo: %d %s %s\n", message.From.ID, message.From.FirstName, message.From.LastName)
			}
		case "/new_request":
			_, _ = bot.SendMessage(tu.Message(tu.ID(message.Chat.ID),
				fmt.Sprintf(`<i>Эта функция еще в разработке</i>`)).WithParseMode("HTML"))
			_, _ = bot.SendSticker(tu.Sticker(tu.ID(message.Chat.ID), tu.FileByID("CAACAgIAAxkBAAOFY3PZDYNCi8QCC6YZmMW0KAgEL1sAAiUMAAKjYThKJoKMYxEN6pwrBA")))

		}
	}, th.AnyCommand())

	//Handle messages from user
	bh.HandleMessage(func(bot *telego.Bot, message telego.Message) {
		user, _ := mdb.GetUser(message.From.ID)
		switch {
		case user == nil:
			_, _ = bot.SendMessage(tu.Message(
				tu.ID(message.Chat.ID), fmt.Sprintf(
					`Добро пожаловать в бот технической поддержки клиентов!
Опишите вашу проблему <b>одним сообщением</b> и мы постараемся ее решить в кратчайшин строки!`),
			).WithParseMode("HTML"))
			if err := mdb.AddUser(message.From); err != nil {
				log.Println("error addUser: ", err)
			}
		case user != nil:
			mdb.IsFirstMessage(user)

		}
	}, th.Not(th.AnyCommand()))

	// Handler channel posts
	bh.HandleChannelPost(func(bot *telego.Bot, message telego.Message) {
		//TODO: make saving channel post to DB

	}, th.AnyChannelPost())

	// Handle supergroup message for a message forwarded form channel
	bh.Handle(func(bot *telego.Bot, update telego.Update) {
		// Send description message
		_, _ = bot.SendMessage(tu.Message(
			tu.ID(update.Message.Chat.ID),
			fmt.Sprintf(
				"<b>Тикет #%d.</b>\nОбращение от %s.\n<code>Для ответа пользователю воспользуйтесь функцией 'Ответить ⤺' на любое сообщение бота</code>",
				update.Message.ForwardFromMessageID, update.Message.From.FirstName),
		).WithParseMode("HTML").WithReplyToMessageID(update.Message.MessageID))
	}, func(update telego.Update) bool {
		return update.Message.From.ID == 777000 &&
			update.Message.IsAutomaticForward &&
			update.Message.Chat.ID == settings.SupergroupID &&
			update.Message.SenderChat.ID == settings.ChannelID
	})

	bh.HandleMessage(func(bot *telego.Bot, message telego.Message) {
		// Send message with inline keyboard
		_, _ = bot.SendMessage(tu.Message(
			tu.ID(message.Chat.ID),
			fmt.Sprintf("%s, уверены что хотите закрыть обращение?", message.From.FirstName),
		).WithReplyToMessageID(message.MessageID).WithReplyMarkup(
			tu.InlineKeyboard(
				tu.InlineKeyboardRow(
					tu.InlineKeyboardButton("ДА").WithCallbackData("close:"+strconv.Itoa(message.MessageThreadID)),
					tu.InlineKeyboardButton("НЕТ").WithCallbackData("close?no"),
				)),
		))
	}, th.CommandEqual("close"),
		func(update telego.Update) bool {
			return update.Message.Chat.ID == settings.SupergroupID
		})

	bh.HandleCallbackQuery(func(bot *telego.Bot, query telego.CallbackQuery) {
		_, _ = bot.SendMessage(tu.Message(
			tu.ID(query.Message.Chat.ID),
			fmt.Sprintf("%s, бывает...", query.Message.From.FirstName),
		).WithReplyToMessageID(query.Message.MessageID),
		)
		// Answer callback query
		_ = bot.AnswerCallbackQuery(tu.CallbackQuery(query.ID).WithText("Done"))
	}, th.AnyCallbackQueryWithMessage(), th.CallbackDataEqualFold("close?no"))

	//Handle CLOSING request
	bh.HandleCallbackQuery(
		func(bot *telego.Bot, query telego.CallbackQuery) {
			request := strings.TrimPrefix(query.Data, "close:")
			user := query.From.Username
			messageID := query.Message.MessageID
			chatID := query.Message.Chat.ID
			//_, _ = bot.SendMessage(tu.Message(
			//	tu.ID(query.Message.Chat.ID),
			//	fmt.Sprintf("Тикет №%s закрыт @%s", request, user),
			//).WithReplyToMessageID(messageID),
			//)
			// Delete inline keyboard
			_, _ = bot.EditMessageReplyMarkup(deleteInlineKeyboard(messageID, chatID))
			editMessageParam := telego.EditMessageTextParams{}
			_, _ = bot.EditMessageText(editMessageParam.WithChatID(tu.ID(chatID)).WithMessageID(messageID).WithText(
				fmt.Sprintf("request #%s closed by @%s", request, user)))

			// Answer callback query
			_ = bot.AnswerCallbackQuery(tu.CallbackQuery(query.ID).WithText(
				fmt.Sprintf("request #%s closed by @%s", request, user)))

			// Delete message
			//deleteMessageParam := telego.DeleteMessageParams{ChatID: tu.ID(chatID), MessageID: messageID}
			//_ = bot.DeleteMessage(&deleteMessageParam)
		},
		th.AnyCallbackQueryWithMessage(), th.CallbackDataPrefix("close:"))

	// Start handling updates
	bh.Start()
}

func deleteInlineKeyboard(messageID int, chatID int64) *telego.EditMessageReplyMarkupParams {
	editMarkupParams := telego.EditMessageReplyMarkupParams{}
	emptyKeyboard := editMarkupParams.WithMessageID(messageID).WithChatID(tu.ID(chatID)).WithReplyMarkup(
		&telego.InlineKeyboardMarkup{InlineKeyboard: make([][]telego.InlineKeyboardButton, 0)})
	return emptyKeyboard
}
