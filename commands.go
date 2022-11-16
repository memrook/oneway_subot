package main

import (
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoutil"
	"log"
)

func SetBotCommands(bot *telego.Bot) {
	var err error

	dcp := telego.DeleteMyCommandsParams{}
	err = bot.DeleteMyCommands(dcp.WithScope(telegoutil.ScopeAllPrivateChats()).WithLanguageCode("ru"))
	err = bot.DeleteMyCommands(dcp.WithScope(telegoutil.ScopeAllPrivateChats()))
	err = bot.DeleteMyCommands(dcp.WithScope(telegoutil.ScopeDefault()).WithLanguageCode("ru"))
	err = bot.DeleteMyCommands(dcp.WithScope(telegoutil.ScopeDefault()))
	err = bot.DeleteMyCommands(dcp.WithScope(telegoutil.ScopeAllChatAdministrators()).WithLanguageCode("ru"))
	err = bot.DeleteMyCommands(dcp.WithScope(telegoutil.ScopeAllChatAdministrators()))
	err = bot.DeleteMyCommands(dcp.WithScope(telegoutil.ScopeAllGroupChats()).WithLanguageCode("ru"))
	err = bot.DeleteMyCommands(dcp.WithScope(telegoutil.ScopeAllGroupChats()))

	//set commands
	defaultBotCommands := []telego.BotCommand{
		{Command: "start", Description: "Начнем, пожалуй"},
	}

	privateBotCommands := []telego.BotCommand{
		{Command: "list_requests", Description: "Список обращений"},
		{Command: "new_request", Description: "Новое обращение"},
		{Command: "close_request", Description: "Закрыть обращение"},
	}

	groupChatsBotCommands := []telego.BotCommand{
		{Command: "count", Description: "Количество открытых тикетов"},
		{Command: "list", Description: "Список открытых тикетов"},
		{Command: "close", Description: "Закрыть тикет"},
	}

	dc := telego.SetMyCommandsParams{}
	pc := telego.SetMyCommandsParams{}
	gcc := telego.SetMyCommandsParams{}

	defaultCommandsParams := dc.WithCommands(defaultBotCommands...).WithScope(telegoutil.ScopeDefault()).WithLanguageCode("ru")
	privateCommandsParam := pc.WithCommands(privateBotCommands...).WithScope(telegoutil.ScopeAllPrivateChats()).WithLanguageCode("ru")
	groupChatsCommandsParam := gcc.WithCommands(groupChatsBotCommands...).WithScope(telegoutil.ScopeAllGroupChats()).WithLanguageCode("ru")

	if err = bot.SetMyCommands(defaultCommandsParams); err != nil {
		log.Println("SetMyCommands failed: ", err)
	}
	if err = bot.SetMyCommands(privateCommandsParam); err != nil {
		log.Println("SetMyCommands failed: ", err)
	}
	if err = bot.SetMyCommands(groupChatsCommandsParam); err != nil {
		log.Println("SetMyCommands failed: ", err)
	}

	var getMyCommands telego.GetMyCommandsParams
	bc, _ := bot.GetMyCommands(getMyCommands.WithScope(telegoutil.ScopeAllPrivateChats()).WithLanguageCode("ru"))
	bc2, _ := bot.GetMyCommands(getMyCommands.WithScope(telegoutil.ScopeAllPrivateChats()))
	bc3, _ := bot.GetMyCommands(getMyCommands.WithScope(telegoutil.ScopeDefault()).WithLanguageCode("ru"))
	bc4, _ := bot.GetMyCommands(getMyCommands.WithScope(telegoutil.ScopeDefault()))
	log.Println("BOT COMMANDS", bc)
	log.Println("BOT COMMANDS", bc2)
	log.Println("BOT COMMANDS", bc3)
	log.Println("BOT COMMANDS", bc4)
}
