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

type Session struct {
	Tomatoes      uint8
	UserID        int64
	MessageID     int
	ExpiresAt     time.Time
	actionID      uint8 // work, rest
	isActionEnded bool
	notifier      bool // true if enabled
}

var UserSession = make(map[int64]Session)
var ch = make(chan Session)
var bot *tgbotapi.BotAPI

var IKBwait = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
	tgbotapi.NewInlineKeyboardButtonData("Start working", "work"),
	tgbotapi.NewInlineKeyboardButtonData("Stop working", "rest"),
))

// Series of notifications when
var IKBreminder = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
	tgbotapi.NewInlineKeyboardButtonData("OK", "ok"),
))

func main() {

	fmt.Println("Started working")
	godotenv.Load()

	bot, _ = tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_API_TOKEN"))
	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		var withoutErrors = true

		// change session
		if update.CallbackQuery != nil {
			switch update.CallbackQuery.Data {
			case "work":
				print("\nWork")
				// if session does not exist yet
				if _, ok := UserSession[update.CallbackQuery.From.ID]; !ok {
					UserSession[update.CallbackQuery.From.ID] = Session{
						Tomatoes:      0,
						UserID:        update.CallbackQuery.From.ID,
						MessageID:     update.CallbackQuery.Message.MessageID,
						ExpiresAt:     time.Now().Add(time.Duration(1 * time.Minute / 5)), //25
						actionID:      1,
						isActionEnded: false,
						notifier:      true,
					}
					print("\n\tSession didn't exists and we've just created it")
				} else {
					UserSession[update.CallbackQuery.From.ID] = Session{
						Tomatoes:      UserSession[update.CallbackQuery.From.ID].Tomatoes,
						UserID:        UserSession[update.CallbackQuery.From.ID].UserID,
						MessageID:     update.CallbackQuery.Message.MessageID,
						ExpiresAt:     time.Now().Add(time.Duration(1 * time.Minute / 5)), // 25
						actionID:      1,
						isActionEnded: false,
						notifier:      true,
					}
					print("\n\tSession already existed. We've just modified it")
				}
			case "rest":
				print("\nRest")
				if _, ok := UserSession[update.CallbackQuery.From.ID]; ok {
					print("\n\tSession exists")
					UserSession[update.CallbackQuery.From.ID] = Session{
						Tomatoes:      0,
						UserID:        update.CallbackQuery.From.ID,
						MessageID:     update.CallbackQuery.Message.MessageID,
						ExpiresAt:     time.Now().Add(time.Duration(1 * time.Minute / 6)), // 5
						actionID:      2,
						isActionEnded: false,
						notifier:      true,
					}

					if v, ok := UserSession[update.CallbackQuery.From.ID]; UserSession[update.CallbackQuery.From.ID].Tomatoes == 4 && ok {
						print("\n\tSession exists and amount of tomatoes is 4. So now will be a big rest")
						v.ExpiresAt = time.Now().Add(time.Duration(1 * time.Minute / 6)) // 15
						UserSession[update.CallbackQuery.From.ID] = v
					} else {
						print("\n\tERROR: Somebody clicked on Rest without new Session or something else happened.")
						withoutErrors = false
					}

				} else {
					m := tgbotapi.NewMessage(update.CallbackQuery.From.ID, "Sorry, now you have to create a new session! Type /start")
					withoutErrors = false
					bot.Send(m)
					print("\n\tSession doesn't exist")

				}

			case "ok":
				print("\nOK button click")
				if v, ok := UserSession[update.CallbackQuery.From.ID]; ok {
					print("\n\tUser session exist")
					v.notifier = false
					print("\n\tDisabled notifier")
					UserSession[update.CallbackQuery.From.ID] = v
					dl := tgbotapi.NewDeleteMessage(update.CallbackQuery.From.ID, update.CallbackQuery.Message.MessageID)
					bot.Request(dl)
					print("\n\tRemoved message with OK button click")
				} else {
					print("\n\tSession doesn't exist, but got OK button click.")
					withoutErrors = false
					dl := tgbotapi.NewDeleteMessage(update.CallbackQuery.From.ID, update.CallbackQuery.Message.MessageID)
					bot.Request(dl)
					print("\n\tRemoved message with OK button click. Continue")

					continue
				}
			default:
				print("\n\tAnother inline button data. Continue checking for a new messages.")
				continue
			}

			msgEdit := genMessage(UserSession[update.CallbackQuery.From.ID])
			msg, err := bot.Send(msgEdit)
			if err != nil {
				print("\n\tERROR while sending message", err)
				fmt.Println(UserSession[update.CallbackQuery.From.ID])
			}

			// Check for new notifications from schedule() and send them
			if withoutErrors {
				print("\n\tStart message changer")
				go MessageChanger(msg, UserSession[update.CallbackQuery.From.ID])
			}

			continue

		}

		if update.Message != nil && update.Message.IsCommand() {
			print("\nGot a new command")
			if update.Message.Command() == "start" {
				print("\n\tStart command")
				var msg tgbotapi.MessageConfig
				if _, ok := UserSession[update.Message.Chat.ID]; ok {
					print("\n\t\tSession exists")
					msg := genMessage(UserSession[update.Message.Chat.ID])
					bot.Send(msg)
					print("\n\t\tSend message with the existing session")
				} else {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Start your new session with the button below")
					msg.ReplyMarkup = IKBwait
					print("\n\t\tSession doesn't exist. Send user message with inline keyboard")
					bot.Send(msg)
				}
			}
		}
	}
}

func MessageChanger(msg tgbotapi.Message, s Session) {
	for time.Now().Before(s.ExpiresAt) {
		print("\nMessage changer cycle")
		// Edit message and get how many minutes are left
		remainsMins := s.ExpiresAt.Sub(time.Now())
		fmt.Println("remains:", int(remainsMins.Minutes()))

		if remainsMins > time.Duration(time.Minute) { // If remains more than 1 minute
			fmt.Println(remainsMins.Minutes(), "minutes left")
			msg := genMessage(s)
			bot.Send(msg)
			time.Sleep(time.Duration(time.Minute))
		} else if remainsMins > time.Duration(5*time.Second) {
			// Modify message with seconds
			fmt.Println(remainsMins.Seconds(), "seconds left")
			msg := genMessageWithSeconds(s)
			bot.Send(msg)
			time.Sleep(5 * time.Second)

		} else if remainsMins > time.Duration(1*time.Second) {
			// Modify message with seconds
			fmt.Println(remainsMins.Seconds(), "seconds left")
			msg := genMessageWithSeconds(s)
			bot.Send(msg)
			time.Sleep(1 * time.Second)

		} else {
			print("\n\tBreak because remainsMins is less than current time")
			break
		}
	}

	// Now user's session is ended
	if v, ok := UserSession[s.UserID]; ok {
		print("\n\tChange user's session parameters")
		v.isActionEnded = true
		v.notifier = true
		v.Tomatoes += 1
		v.actionID = 3 - v.actionID
		UserSession[s.UserID] = v
	}

	// Notify user for 5 times while he won't click OK
	for i := 0; i < 5; i++ {
		// if user clicked the message just break
		if !UserSession[s.UserID].notifier {
			print("\n\tNotifier disabled. Break")
			break
		}
		print("\n\tNotificate uesr")
		ss := tgbotapi.NewMessage(msg.Chat.ID, "Your time has ended!")
		ss.ReplyMarkup = IKBreminder
		msg, err := bot.Send(ss)
		if err != nil {
			print("\n\tGot error while sending message")
			panic(err)

		}
		print("\n\tSchedule delete for notification message")
		scheduleDelete(msg, time.Duration(30*time.Second/8))

	}

	//
	// Send message about time expiration
	//
}

// genEditMessage generates message for current state
func genMessage(s Session) (editMsg tgbotapi.EditMessageTextConfig) {
	print("\nGeneration of the new message")
	var msgText string
	var msgMarkup tgbotapi.InlineKeyboardMarkup

	// if timer has been ended and now user have to click something to continue
	if s.isActionEnded {
		print("\n\tAction ended")
		// work's ended
		if s.actionID == 1 {
			print("\n\t\tWork ended")
			// Rest for 5 mins
			msgText = "You have worked for 25 minutes and now is a time to take good short rest for. Try to change your environment!\n\n Click the button below to start rest"
			msgMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Rest", "rest"),
				),
			)
			// If tomatos number is 4, then change message and rest for 15 mins
			if s.Tomatoes == 4 {
				print("\n\t\tAmount of tomatoes is 4. User should have a long rest")
				msgText = "After great job comes great rest. Now you will rest for 15 minutes. Remember to take care of your body. If you've been sitting all this time, now is a great time for physical exercises!\n\nRest for 15 minutes after clicking the button below"
				msgMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("Long rest", "rest"),
					),
				)
			}
		} else { // Rest has been ended
			print("\n\tRest ended")
			msgText = "It's time for a next portion of job. Wish you have productive time!\n\nStart work after click the button"
			msgMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Work", "work"),
				),
			)
			if s.Tomatoes == 8 { // The end of a session
				print("\n\t\tSession ended, as user have collected 8 tomatoes")
				msgText = "Your session has been ended! Nice job. You did really well! See you next time. Bye"
				msgMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("Bye", "bye"),
					),
				)
			}
		}

		if s.Tomatoes == 1 {
			print("\n\tUser got one tomato")
			msgText += "You earned your first tomato in the session. Here it is:\n🍅"
		} else {
			print("\n\tUser hove not 1 tomato")
			msgText += "Here are your tomatoes:\n" + strings.Repeat("🍅", int(s.Tomatoes))
		}

		print("\nCreate edit message confing and return it")
		editMsg = tgbotapi.NewEditMessageTextAndMarkup(s.UserID, s.MessageID, msgText, msgMarkup)
		return
	}

	print("\nAction haven't ended")
	// if work
	if s.actionID == 1 {
		print("\n\tUpdate work time")
		msgText = fmt.Sprintf("Work time! %d minutes left!", int(s.ExpiresAt.Sub(time.Now()).Minutes()))
	}

	// if small rest
	if s.actionID == 2 {
		print("\n\tUpdate rest time")
		msgText = fmt.Sprintf("Rest time. %d minutes left!", int(s.ExpiresAt.Sub(time.Now()).Minutes()))
	}

	// if big rest
	if s.actionID == 2 {
		print("\n\tUpdate big rest time")
		msgText = fmt.Sprintf("Time for big rest! %d minutes left!", int(s.ExpiresAt.Sub(time.Now()).Minutes()))
	}

	print("\nCreate edit message confing and return it")
	editMsg = tgbotapi.NewEditMessageText(s.UserID, s.MessageID, msgText)

	return
}

func genMessageWithSeconds(s Session) (editMsg tgbotapi.EditMessageTextConfig) {

	print("\nGenerate message to update remained seconds")
	var msgText string

	// if work
	if s.actionID == 1 {
		print("\n\tUpdate work's seconds")
		msgText = fmt.Sprintf("Work time! %d seconds left!", int(s.ExpiresAt.Sub(time.Now()).Seconds()))
	}

	// if small rest
	if s.actionID == 2 {
		print("\n\tUpdate rest's seconds")
		msgText = fmt.Sprintf("Rest time. %d seconds left!", int(s.ExpiresAt.Sub(time.Now()).Seconds()))
	}

	// if big rest
	if s.actionID == 2 {
		print("\n\tUpdate big rest's seconds")
		msgText = fmt.Sprintf("Time for big rest! %d seconds left!", int(s.ExpiresAt.Sub(time.Now()).Seconds()))
	}

	editMsg = tgbotapi.NewEditMessageText(s.UserID, s.MessageID, msgText)
	print("\n\tEdit message and return")
	return
}

func scheduleDelete(msg tgbotapi.Message, d time.Duration) {
	print("\n\tScheduled deletion in ", d.Seconds(), " seconds")
	time.Sleep(d)
	dl := tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID)
	bot.Request(dl)

}

// type UserNotification struct {
// 	UserID          int64
// 	TomatoNumber    uint8
// 	IsWork          bool
// 	RemindInMinutes uint16
// }

// var ch = make(chan UserNotification)
// var bot *tgbotapi.BotAPI

// func main() {

// 	fmt.Println("Started working")
// 	godotenv.Load()

// 	bot, _ = tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_API_TOKEN"))
// 	bot.Debug = false
// 	log.Printf("Authorized on account %s", bot.Self.UserName)

// 	u := tgbotapi.NewUpdate(0)
// 	u.Timeout = 60

// 	updates := bot.GetUpdatesChan(u)

// 	// Check for new notifications from schedule() and send them
// 	go notificator()

// 	for update := range updates {

// 		if update.CallbackQuery != nil {
// 			userID := update.CallbackQuery.From.ID

// 			switch update.CallbackQuery.Data {
// 			case "work":
// 				go schedule(UserNotification{
// 					UserID:          userID,
// 					TomatoNumber:    UsersTomatoesNumber(userID),
// 					IsWork:          true,
// 					RemindInMinutes: UserWorkTime(userID),
// 				})

// 			case "rest":
// 				go schedule(UserNotification{
// 					UserID:          userID,
// 					TomatoNumber:    UsersTomatoesNumber(userID),
// 					IsWork:          false,
// 					RemindInMinutes: UserRestTime(userID),
// 				})
// 			}

// 			msg := tgbotapi.NewMessage(userID, "Timer has been started!")
// 			bot.Send(msg)
// 			continue
// 		}

// 		if update.Message != nil && update.Message.IsCommand() {
// 			userID := update.Message.Chat.ID
// 			var msg tgbotapi.MessageConfig

// 			if update.Message.Text == "/start" {
// 				msg = tgbotapi.NewMessage(userID, "Press the button to start!")
// 				msg.ReplyMarkup = UserInlineKeyboard(userID)
// 			} else {
// 				msg = tgbotapi.NewMessage(userID, "Unknow command. Please type /start to get your buttons.")
// 			}
// 			bot.Send(msg)
// 			continue
// 		}
// 	}
// }

// func notificator() {
// 	for {
// 		select {
// 		case un := <-ch:
// 			var (
// 				ikb           tgbotapi.InlineKeyboardMarkup
// 				messageToUser string
// 			)

// 			if un.IsWork {
// 				ikb = tgbotapi.NewInlineKeyboardMarkup(
// 					tgbotapi.NewInlineKeyboardRow(
// 						tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s(%d)", "Rest", UserRestTime(un.UserID)), "rest"),
// 					),
// 				)
// 				messageToUser = "Current part of your work time has ended. Well done!\n\nHere are the tomatoes you have collected:\n" + strings.Repeat("🍅", int(un.TomatoNumber)) + "\n\nPress the button to take a productive rest. Try to change your environment for this time.\n\n"

// 			} else {
// 				ikb = tgbotapi.NewInlineKeyboardMarkup(
// 					tgbotapi.NewInlineKeyboardRow(
// 						tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s(%d)", "Work", UserWorkTime(un.UserID)), "work"),
// 					),
// 				)
// 				messageToUser = "Your rest has ended. Please touch the button below to start another productive working time!"
// 			}

// 			msg := tgbotapi.NewMessage(un.UserID, messageToUser)
// 			msg.ReplyMarkup = ikb
// 			bot.Send(msg)

// 			fmt.Printf("\nSend message to the user with ID %d\n\nTomato number: %d\nIs work: %v\nReminded in %d minutes\n\nChange user's status to other and increment the number of tomatoes.", un.UserID, un.TomatoNumber, un.IsWork, un.RemindInMinutes)
// 		}

// 	}
// }

// func schedule(un UserNotification) {
// 	time.Sleep(time.Duration(un.RemindInMinutes) * time.Minute)
// 	ch <- un
// }

// func UsersTomatoesNumber(userID int64) uint8 {
// 	return 5
// }

// func UserRestTime(userID int64) uint16 {
// 	// Here I will get user's short and long rest time from DB
// 	longBreak := uint16(15)
// 	shortBreak := uint16(5)

// 	if UsersTomatoesNumber(userID) == 4 {
// 		return longBreak
// 	}
// 	return shortBreak
// }

// func UserWorkTime(userID int64) uint16 {
// 	// Here I will get user's working time from DB
// 	workTime := uint16(25)
// 	return workTime
// }

// func isWork(userID int64) bool {
// 	// Check in the database if the user chould be work or rest
// 	return false
// }

// func UserInlineKeyboard(userID int64) tgbotapi.InlineKeyboardMarkup {
// 	ikbData := "Rest"
// 	isWork := isWork(userID)
// 	time := UserRestTime(userID)

// 	if isWork {
// 		ikbData = "Work"
// 		time = UserWorkTime(userID)
// 	}

// 	var ikbText = fmt.Sprintf("%s(%d)", ikbData, time)

// 	return tgbotapi.NewInlineKeyboardMarkup(
// 		tgbotapi.NewInlineKeyboardRow(
// 			tgbotapi.NewInlineKeyboardButtonData(ikbText, strings.ToLower(ikbData)),
// 		),
// 	)
// }
