package main

import (
	"fmt"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"go.mongodb.org/mongo-driver/mongo"
	mdb "goland/oneWaySupportBot/mongodb"
	"gopkg.in/gookit/color.v1"
	"log"
	"strconv"
	"strings"
)

func handlePrivateCommands(bot *telego.Bot, message telego.Message) {
	switch message.Text {
	case "/start":
		user, ok := mdb.GetUser(message.From.ID)
		if ok != mongo.ErrNoDocuments {
			_, _ = bot.SendMessage(tu.Message(
				tu.ID(message.Chat.ID), fmt.Sprintf(
					"С возвращением, %s!\n Опишите вашу проблему "+
						"<b>одним сообщением</b> и мы постараемся ее решить в кратчайшин строки!", user.FirstName),
			).WithParseMode("HTML"))
		} else {
			_, _ = bot.SendMessage(tu.Message(
				tu.ID(message.Chat.ID), fmt.Sprintf(
					"Добро пожаловать в бот технической поддержки клиентов!\n"+
						"Опишите вашу проблему <b>одним сообщением</b> и мы "+
						"постараемся ее решить в кратчайшин строки!"),
			).WithParseMode("HTML"))
			newUser := mdb.User{}
			err := newUser.New(message.From)
			if err != nil {
				log.Println("error addUser: ", err)
			}
			color.Yellow.Printf("Add user to Mongodb: %d %s %s\n", newUser.ID, newUser.FirstName, newUser.LastName)
		}
	case "/close":
		//TODO send a quality question
		threadID := mdb.GetThreadIDbyUsername(message.Chat.Username)
		_, _ = bot.SendMessage(tu.Message(
			tu.ID(message.Chat.ID),
			fmt.Sprintf("%s, уверены что хотите закрыть обращение?", message.From.FirstName),
		).WithReplyToMessageID(message.MessageID).WithReplyMarkup(
			tu.InlineKeyboard(
				tu.InlineKeyboardRow(
					tu.InlineKeyboardButton("ДА ✅").WithCallbackData("close:"+strconv.Itoa(threadID)),
					tu.InlineKeyboardButton("НЕТ ❌").WithCallbackData("close?no"),
				)),
		))
	default:
		_, _ = bot.SendMessage(tu.Message(tu.ID(message.Chat.ID),
			fmt.Sprintf(`<i>Эта функция еще в разработке</i>`)).WithParseMode("HTML"))
		//_, _ = bot.SendSticker(tu.Sticker(
		//	tu.ID(message.Chat.ID),
		//	tu.FileByID("CAACAgIAAxkBAAOFY3PZDYNCi8QCC6YZmMW0KAgEL1sAAiUMAAKjYThKJoKMYxEN6pwrBA"),
		//))

	}
}

func handleGroupCommands(bot *telego.Bot, message telego.Message) {
	switch message.Text {
	case "/close@" + botUser.Username:
		// Send message with inline keyboard
		_, _ = bot.SendMessage(tu.Message(
			tu.ID(message.Chat.ID),
			fmt.Sprintf("%s, уверены что хотите закрыть обращение?", message.From.FirstName),
		).WithReplyToMessageID(message.MessageID).WithReplyMarkup(
			tu.InlineKeyboard(
				tu.InlineKeyboardRow(
					tu.InlineKeyboardButton("ДА ✅").WithCallbackData("close:"+strconv.Itoa(message.MessageThreadID)),
					tu.InlineKeyboardButton("НЕТ ❌").WithCallbackData("close?no"),
				)),
		))
	default:
		_, _ = bot.SendMessage(tu.Message(tu.ID(message.Chat.ID),
			fmt.Sprintf(`<i>Эта функция еще в разработке 🛠️</i>`)).
			WithParseMode("HTML").WithReplyToMessageID(message.MessageThreadID))
	}
}

func handlePrivateMessage(bot *telego.Bot, message telego.Message) {
	user, _ := mdb.GetUser(message.From.ID)
	switch {
	case user == nil:
		_, _ = bot.SendMessage(tu.Message(
			tu.ID(message.Chat.ID), fmt.Sprintf(
				"Добро пожаловать в бот технической поддержки клиентов!\n"+
					"Опишите вашу проблему <b>одним сообщением</b> "+
					"и мы постараемся ее решить в кратчайшин строки!"),
		).WithParseMode("HTML"))

		if err := mdb.User.New(mdb.User{}, message.From); err != nil {
			log.Println("failed to addUser: ", err)
		}
	case user != nil:
		// if a message is the first >> write to db and forward the request to the channel
		chatID := mdb.Chat.FindChatIDByUserID(mdb.Chat{}, message.From.ID)
		if chatID == 0 {
			res, _ := bot.SendMessage(tu.Message(
				tu.ID(settings.ChannelID), fmt.Sprintf(
					"<b>Обращение #%d \nот @%s</b>\n\n%s\n",
					message.Chat.ID,
					message.From.Username,
					message.Text,
				),
			).WithParseMode("HTML"))

			err := mdb.NewChat(res.MessageID, &message)
			if err != nil {
				log.Println("failed NewChat due ERR:", err)
			}

			//Send the user a confirmation of the registration of a new chat
			_, _ = bot.SendMessage(tu.Message(
				tu.ID(message.Chat.ID), fmt.Sprintf(
					"<i>Ваша заявка принята!</i>"),
			).WithParseMode("HTML"))
		} else {
			_, err := bot.CopyMessage(tu.CopyMessage(
				tu.ID(settings.SupergroupID),
				tu.ID(message.Chat.ID),
				message.MessageID,
			).WithReplyToMessageID(chatID))

			if err != nil {
				if strings.Contains(err.Error(), "replied message not found") {
					//
					_, _ = bot.SendMessage(tu.Message(
						tu.ID(message.Chat.ID), fmt.Sprintf(
							"Упс... Не могу найти ваше обращение 🥺\n"+
								"Опишите пожалуйста вашу проблему "+
								"и мы постараемся решить ее как можно быстрее!"),
					).WithParseMode("HTML"))
					err = mdb.CloseAllChats(message.From.ID)
					if err != nil {
						color.Red.Println("failed to close all chats due: ", err)
					}
					break
				} else {
					color.Red.Println("failed to send message due: ", err)
					break
				}
			}
			if err = mdb.AddMessage(chatID, &message); err != nil {
				log.Println(err)
			}

		}
		//_, _ = bot.ForwardMessage(&telego.ForwardMessageParams{ChatID: tu.ID(Settings.ChannelID), FromChatID: tu.ID(message.Chat.ID), MessageID: message.MessageID})

	}
}

func handleGroupPostFromChannel(bot *telego.Bot, message telego.Message) {
	// Send description message
	_, _ = bot.SendMessage(tu.Message(
		tu.ID(message.Chat.ID),
		fmt.Sprintf("<code>Для ответа пользователю воспользуйтесь функцией 'Ответить ⤺' на любое сообщение бота</code>"),
	).WithReplyToMessageID(message.MessageID).WithParseMode("HTML").WithDisableNotification())
	// Update chat ID Обновление
	err := mdb.UpdateChatID(message.ForwardFromMessageID, message.MessageID)
	if err != nil {
		color.Red.Println("failed update chatID in post: ", err)
	} else {
		if err == mdb.AddMessage(message.MessageID, &message) {
			log.Println(err)
		}
	}
}

func handleReplyGroupMessage(bot *telego.Bot, message telego.Message) {
	chatID, _ := mdb.GetUserByChatID(message.MessageThreadID)
	if chatID != 0 {
		_, _ = bot.SendMessage(tu.Message(
			tu.ID(chatID), message.Text,
		).WithParseMode("HTML"))

		if err := mdb.AddMessage(message.MessageThreadID, &message); err != nil {
			log.Println(err)
		}
	}
}

func handleCallbackQuery(bot *telego.Bot, query telego.CallbackQuery) {
	user := query.From.Username
	messageID := query.Message.MessageID
	chatID := query.Message.Chat.ID

	switch {
	case strings.HasPrefix(query.Data, "close:"):

		threadID, _ := strconv.Atoi(strings.TrimPrefix(query.Data, "close:"))

		if err := mdb.CloseRequest(threadID); err != nil {
			color.Red.Println("failed to CloseRequest and Reply")
		} else {
			_, _ = bot.EditMessageReplyMarkup(deleteInlineKeyboard(messageID, chatID))
			editMessageParam := telego.EditMessageTextParams{}
			_, _ = bot.EditMessageText(editMessageParam.WithChatID(tu.ID(chatID)).WithMessageID(messageID).WithText(
				fmt.Sprintf("request #%d closed by @%s", threadID, user)))
		}
	case query.Data == "close?no":

		deleteMessageParam := telego.DeleteMessageParams{ChatID: tu.ID(chatID), MessageID: messageID}
		_ = bot.DeleteMessage(&deleteMessageParam)
	}
}

//EXAMPLES:

// Answer callback query
//_ = bot.AnswerCallbackQuery(tu.CallbackQuery(query.ID).WithText(
//	fmt.Sprintf("request #%s closed by @%s", request, user)))

// Delete message
//deleteMessageParam := telego.DeleteMessageParams{ChatID: tu.ID(chatID), MessageID: messageID}
//_ = bot.DeleteMessage(&deleteMessageParam)
