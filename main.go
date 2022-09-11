package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type UserNotification struct {
	UserID          int64
	TomatoNumber    uint8
	IsWork          bool
	RemindInMinutes uint16
}

var ch = make(chan UserNotification)
var bot *tgbotapi.BotAPI

func main() {

	fmt.Println("Started working")
	godotenv.Load()

	bot, _ = tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_API_TOKEN"))
	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	// Check for new notifications from schedule() and send them
	go notificator()

	for update := range updates {

		if update.CallbackQuery != nil {
			userID := update.CallbackQuery.From.ID

			switch update.CallbackQuery.Data {
			case "work":
				go schedule(UserNotification{
					UserID:          userID,
					TomatoNumber:    UsersTomatoesNumber(userID),
					IsWork:          true,
					RemindInMinutes: UserWorkTime(userID),
				})

			case "rest":
				go schedule(UserNotification{
					UserID:          userID,
					TomatoNumber:    UsersTomatoesNumber(userID),
					IsWork:          false,
					RemindInMinutes: UserRestTime(userID),
				})
			}

			msg := tgbotapi.NewMessage(userID, "Timer has been started!")
			bot.Send(msg)
			continue
		}

		if update.Message != nil && update.Message.IsCommand() {
			userID := update.Message.Chat.ID
			var msg tgbotapi.MessageConfig

			if update.Message.Text == "/start" {
				msg = tgbotapi.NewMessage(userID, "Press the button to start!")
				msg.ReplyMarkup = UserInlineKeyboard(userID)
			} else {
				msg = tgbotapi.NewMessage(userID, "Unknow command. Please type /start to get your buttons.")
			}
			bot.Send(msg)
			continue
		}
	}
}

func notificator() {
	for {
		select {
		case un := <-ch:
			var (
				ikb           tgbotapi.InlineKeyboardMarkup
				messageToUser string
			)

			if un.IsWork {
				ikb = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s(%d)", "Rest", UserRestTime(un.UserID)), "rest"),
					),
				)
				messageToUser = "Current part of your work time has ended. Well done!\n\nHere are the tomatoes you have collected:\n" + strings.Repeat("🍅", int(un.TomatoNumber)) + "\n\nPress the button to take a productive rest. Try to change your environment for this time.\n\n"

			} else {
				ikb = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s(%d)", "Work", UserWorkTime(un.UserID)), "work"),
					),
				)
				messageToUser = "Your rest has ended. Please touch the button below to start another productive working time!"
			}

			msg := tgbotapi.NewMessage(un.UserID, messageToUser)
			msg.ReplyMarkup = ikb
			bot.Send(msg)

			fmt.Printf("\nSend message to the user with ID %d\n\nTomato number: %d\nIs work: %v\nReminded in %d minutes\n\nChange user's status to other and increment the number of tomatoes.", un.UserID, un.TomatoNumber, un.IsWork, un.RemindInMinutes)
		}

	}
}

func schedule(un UserNotification) {
	time.Sleep(time.Duration(un.RemindInMinutes) * time.Second)
	ch <- un
}

func UsersTomatoesNumber(userID int64) uint8 {
	return 5
}

func UserRestTime(userID int64) uint16 {
	// Here I will get user's short and long rest time from DB
	longBreak := uint16(15)
	shortBreak := uint16(5)

	if UsersTomatoesNumber(userID) == 4 {
		return longBreak
	}
	return shortBreak
}

func UserWorkTime(userID int64) uint16 {
	// Here I will get user's working time from DB
	workTime := uint16(15)
	return workTime
}

func isWork(userID int64) bool {
	// Check in the database if the user chould be work or rest
	return false
}

func UserInlineKeyboard(userID int64) tgbotapi.InlineKeyboardMarkup {
	ikbData := "Rest"
	isWork := isWork(userID)
	time := UserRestTime(userID)

	if isWork {
		ikbData = "Work"
		time = UserWorkTime(userID)
	}

	var ikbText = fmt.Sprintf("%s(%d)", ikbData, time)

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(ikbText, strings.ToLower(ikbData)),
		),
	)
}
