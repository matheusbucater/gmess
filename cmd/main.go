package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/matheusbucater/gmess/internal/db/sqlc"
	"github.com/matheusbucater/gmess/internal/feat"
	"github.com/matheusbucater/gmess/internal/feat/notifications"
	"github.com/matheusbucater/gmess/internal/feat/todos"
	"github.com/matheusbucater/gmess/internal/utils"

	_ "modernc.org/sqlite"
)

func showMessages(order string, sort string) error {
	ctx := context.Background()
	db, err := utils.DbConnect(ctx)
	if err != nil {
		return err
	}

	queries := sqlc.New(db)

	messages := []sqlc.Message{}
	switch order {
	case "created_at":
		if sort == "ASC" {
			messages, err = queries.GetMessagesOrderByCreatedAtASC(ctx)
		} else {
			messages, err = queries.GetMessagesOrderByCreatedAtDESC(ctx)
		}
	case "updated_at":
		if sort == "ASC" {
			messages, err = queries.GetMessagesOrderByUpdatedAtASC(ctx)
		} else {
			messages, err = queries.GetMessagesOrderByUpdatedAtDESC(ctx)
		}
	case "text":
		if sort == "ASC" {
			messages, err = queries.GetMessagesOrderByTextASC(ctx)
		} else {
			messages, err = queries.GetMessagesOrderByTextDESC(ctx)
		}
	}
	if err != nil {
		return err
	}

	messagesCount := len(messages)

	if messagesCount > 0 {
		fmt.Printf("(order by: '%s' %s)\n\n", order, sort)
	}
	
	var sb strings.Builder
	sb.WriteString("You have ")
	sb.WriteString(strconv.Itoa(messagesCount))
	sb.WriteString(" message")
	
	if messagesCount <= 0 {
		sb.WriteString("s")
	} else if messagesCount == 1 {
		sb.WriteString("\n")
	} else {
		sb.WriteString("s\n")
	}
	fmt.Println(sb.String())

	for _, message := range messages {
		features, err := queries.GetFeaturesByMessageId(ctx, message.ID)
		if err != nil {
			return err
		}

		fmt.Printf("(%d) %s", message.ID, message.Text)

		featureNames := []string{}
		for _, feat := range features {
			if feat.Count == 0 { continue }
			featureNames = append(featureNames, feat.FeatureName[:3])
		}
		fmt.Printf("[%s]\n", strings.Join(featureNames, ","))
	}
	return nil
}

func showMessageDetails(id int64) error {
	ctx := context.Background()
	db, err := utils.DbConnect(ctx)
	if err != nil {
		return err
	}

	queries := sqlc.New(db)

	exists, err := queries.MessageExists(ctx, id)
	if err != nil {
		return err
	}
	if exists == 0 {
		return errors.New("Invalid message ID")
	}

	message, err := queries.GetMessageById(ctx, id)
	if err != nil {
		return err
	}

	fmt.Println("Message details:")
	fmt.Printf(
		"\tid: %d\n\ttext: %s\n\tcreated_at: %s\n\tupdated_at: %s", 
		message.ID, message.Text,
		utils.LocalizeDateTime(message.CreatedAt),
		utils.LocalizeDateTime(message.UpdatedAt),
	)

	features, err := queries.GetFeaturesByMessageId(ctx, message.ID)
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString("\n\tfeatures: ")
	for i, feat := range features {
		if feat.Count == 0 { continue }
		sb.WriteString(feat.FeatureName)
		if i == len(features) - 2 { sb.WriteString(" and ") }
		if i < len(features) - 2 { sb.WriteString(", ") } 
	}
	fmt.Println(sb.String())
	return nil
}

func createMessage(message string) (sqlc.Message, error) {
	ctx := context.Background()
	db, err := utils.DbConnect(ctx)
	if err != nil {
		return sqlc.Message{}, err
	}

	queries := sqlc.New(db)

	newMessage, err := queries.CreateMessage(ctx, message)
	if err != nil {
		return sqlc.Message{}, err
	}
	
	return newMessage, nil
}

func updateMessage(id int64, message string) (sqlc.Message, error) {
	ctx := context.Background()
	db, err := utils.DbConnect(ctx)
	if err != nil {
		return sqlc.Message{}, err
	}

	queries := sqlc.New(db)

	exists, err := queries.MessageExists(ctx, id)
	if (exists == 0) {
		return sqlc.Message{}, errors.New("Invalid message ID")
	}
	
	updatedMessage, err := queries.UpdateMessage(ctx, sqlc.UpdateMessageParams{ ID: id, Text: message })
	if err != nil {
		return sqlc.Message{}, err
	}
	
	return updatedMessage, nil
}

func deleteMessage(id int64) error {
	ctx := context.Background()
	db, err := utils.DbConnect(ctx)
	if err != nil {
		return err
	}

	queries := sqlc.New(db)

	exists, err := queries.MessageExists(ctx, id)
	if (exists == 0) {
		return errors.New("Invalid message ID")
	}
	
	if err = queries.DeleteMessage(ctx, id); err != nil {
		return err
	}

	return nil
}

func main() {
	time.Local, _ = time.LoadLocation("America/Sao_Paulo")
	triggerAtDLayout := "02/01/06 15-04-05" // "DD/MM/YY HH-MM-SS"
	triggerAtTLayout := "15-04-05" // "HH-MM-SS"

	helloCmd := flag.NewFlagSet("hello", flag.ExitOnError)
	helloNameFlag := helloCmd.String("name", "", "name to be helloed")

	showCmd := flag.NewFlagSet("show", flag.ExitOnError)
	showFeaturesFlag := showCmd.Bool("feat", false, "show available features")
	showIdFlag := showCmd.Int64("id", -1, "specify message id to show details")
	showOrderFlag := showCmd.String("order", "created_at", "order by: 'created_at', 'updated_at' or 'title'")
	showDescFlag := showCmd.Bool("desc", false, "retrieve messages in descending order")

	createCmd := flag.NewFlagSet("create", flag.ExitOnError)
	createMessageFlag := createCmd.String("message", "", "message to be created")

	updateCmd := flag.NewFlagSet("update", flag.ExitOnError)
	updateIdFlag := updateCmd.Int64("id", -1, "id of the message to be updated")
	updateMessageFlag := updateCmd.String("message", "", "updated message")

	deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
	deleteIdFlag := deleteCmd.Int64("id", -1, "id of the message to be deleted")

	notifCmd := flag.NewFlagSet("notif", flag.ExitOnError)
	notifActionFlag := notifCmd.String("a", "r", "action:\n\t\"c\" create,\n\t\"r\" read,\n\t\"u\" update,\n\t\"d\" delete")
	notifRecurringFlag := notifCmd.Bool("recur", false, "use recurring notification type")
	notifMsgIdFlag := notifCmd.Int64("msgId", -1, "message id")
	notifTriggerAtFlag := notifCmd.String("triggerAt", "", "trigger notification at\nlayout: DD/MM/YY HH-MM-SS or HH-MM-SS")
	notifWeekDaysFlag := notifCmd.String("weekDays", "", "week days that trigger the notification\n(su,mo,tu,we,th,fr,sa)")
	notifNotIdFlag := notifCmd.Int64("notId", -1, "notification id")
	// TODO: add support to order by trigger_at
	notifOrderFlag := notifCmd.String("order", "created_at", "order by: 'created_at', 'updated_at' or 'type'")
	notifDescFlag := notifCmd.Bool("desc", false, "retrieve notifications in descending order")

	todoCmd := flag.NewFlagSet("todo", flag.ExitOnError)
	todoActionFlag := todoCmd.String("a", "r", "action:\n\t\"c\" create,\n\t\"r\" read,\n\t\"u\" update,\n\t\"d\" delete")
	todoMsgIdFlag := todoCmd.Int64("msgId", -1, "message id")
	todoTodIdFlag := todoCmd.Int64("todId", -1, "todo id")
	todoStatusFlag := todoCmd.String("status", "", "todo status\n(pending,done)")
	todoOrderFlag := todoCmd.String("order", "created_at", "order by: 'created_at', 'updated_at' or 'status'")
	todoDescFlag := todoCmd.Bool("desc", false, "retrieve todos in descending order")

	if len(os.Args) < 2 {
		fmt.Println("expected 'hello', 'show', 'create', 'update', 'delete' or 'notify' subcommand.")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "hello":
		err := helloCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("error parsing cli args: %s\n", err)
			os.Exit(1)
		}
		utils.EnforceRequiredFlags(helloCmd, []string{"name"})

		fmt.Printf("Hello %s!\n", *helloNameFlag)
		if err = notifications.Notify(); err != nil {
			fmt.Printf("error notifying user: %s\n", err)
			os.Exit(1)
		}
	case "show":
		err := showCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("error parsing cli args: %s\n", err)
			os.Exit(1)
		}

		if *showFeaturesFlag == true {
			if err = feat.ShowFeatures(); err != nil {
				fmt.Printf("error showing features: %s\n", err)
				os.Exit(1)
			}
		} else {
			if *showIdFlag != -1 {
				if err = showMessageDetails(*showIdFlag); err != nil {
					fmt.Printf("error showing message details: %s\n", err)
					os.Exit(1)
				}
				os.Exit(0)
			} else {
				sort := "ASC"
				if *showDescFlag == true {
					sort = "DESC"
				}

				if !slices.Contains([]string{"created_at", "updated_at", "text"}, strings.ToLower(*showOrderFlag)) {
					fmt.Println("invalid value for '-order' flag")
					showCmd.Usage()
					os.Exit(1)
				}
				if err = showMessages(strings.ToLower(*showOrderFlag), sort); err != nil {
					fmt.Printf("error displaying messages: %s\n", err)
					os.Exit(1)
				}
			}

		}
	case "create":
		err := createCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("error parsing cli args: %s\n", err)
		}
		utils.EnforceRequiredFlags(createCmd, []string{"message"})
		newMessage, err := createMessage(*createMessageFlag)
		if err != nil {
			fmt.Printf("error creating new message: %s\n", err)
		}
		fmt.Printf("message created: (%d) \"%s\".\n", newMessage.ID, newMessage.Text)
	case "update":
		err := updateCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("error parsing cli args: %s\n", err)
			os.Exit(1)
		}
		utils.EnforceRequiredFlags(updateCmd, []string{"id", "message"})
		updatedMessage, err := updateMessage(*updateIdFlag, *updateMessageFlag)
		if err != nil {
			fmt.Printf("error updating message: %s\n", err)
			os.Exit(1)
		}
		fmt.Println(updatedMessage)
	case "delete":
		err := deleteCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("error parsing cli args: %s\n", err)
			os.Exit(1)
		}
		utils.EnforceRequiredFlags(deleteCmd, []string{"id"})
		if err = deleteMessage(*deleteIdFlag); err != nil {
			fmt.Printf("error deleting message: %s\n", err)
			os.Exit(1)
		}
		fmt.Println("Message deleted.")
	case "notif":
		exists, err := feat.FeatureExists(feat.E_notifications_feature)
		if err != nil {
			fmt.Printf("error checking feature existence: %s\n", err)
			os.Exit(1)
		}
		if !exists {
			fmt.Println("feature \"notif\" not available")
			os.Exit(0)
		}
		if err = notifCmd.Parse(os.Args[2:]); err != nil {
			fmt.Printf("error parsing cli args: %s\n", err)
			os.Exit(1)
		}

		switch *notifActionFlag {
		case "c":
			if *notifRecurringFlag == true {
				utils.EnforceRequiredFlags(notifCmd, []string{"msgId", "weekDays", "triggerAt"})
				weekDays, err := utils.ParseWeekDays(*notifWeekDaysFlag)
				if err != nil {
					fmt.Printf("errror parsing weekDays: %s\n", err)
					os.Exit(1)
				}
				triggerAt, err := time.Parse(triggerAtTLayout, *notifTriggerAtFlag)
				if err != nil {
					fmt.Printf("errror parsing triggerAtT date: %s\n", err)
					os.Exit(1)
				}
				if err = notifications.CreateRecurringNotification(*notifMsgIdFlag, weekDays, triggerAt); err != nil {
					fmt.Printf("error creating notification: %s\n", err)
					os.Exit(1)
				}
				os.Exit(0)
			} else {
				utils.EnforceRequiredFlags(notifCmd, []string{"msgId", "triggerAt"})
				triggerAt, err := time.Parse(triggerAtDLayout, *notifTriggerAtFlag)
				triggerAt = time.Date(
					triggerAt.Year(), triggerAt.Month(), triggerAt.Day(), 
					triggerAt.Hour(), triggerAt.Minute(), triggerAt.Second(),
					0, time.Local,
				)
				if err = notifications.CreateSimpleNotification(*notifMsgIdFlag, triggerAt); err != nil {
					fmt.Printf("error creating notification: %s\n", err)
					os.Exit(1)
				}
				os.Exit(0)
			}
		case "r":
			if *notifNotIdFlag != -1 {
				if err = notifications.ShowNotificationDetails(*notifNotIdFlag); err != nil {
					fmt.Printf("error showing notification details: %s\n", err)
					os.Exit(1)
				}
			} else {
				sort := "ASC"
				if *notifDescFlag == true {
					sort = "DESC"
				}

				if !slices.Contains([]string{"created_at", "updated_at", "type"}, strings.ToLower(*notifOrderFlag)) {
					fmt.Println("invalid value for '-order' flag")
					notifCmd.Usage()
					os.Exit(1)
				}
				if err = notifications.ShowNotifications(strings.ToLower(*notifOrderFlag), sort); err != nil {
					fmt.Printf("error showing notifications: %s\n", err)
					os.Exit(1)
				}
			}
		case "u":
			utils.EnforceRequiredFlags(notifCmd, []string{"notId", "triggerAt"})
			triggerAt, err := time.Parse(triggerAtDLayout, *notifTriggerAtFlag)
			triggerAt = time.Date(
				triggerAt.Year(), triggerAt.Month(), triggerAt.Day(), 
				triggerAt.Hour(), triggerAt.Minute(), triggerAt.Second(),
				0, time.Local,
			)
			if err = notifications.UpdateNotification(*notifNotIdFlag, *notifTriggerAtFlag, *notifWeekDaysFlag); err != nil {
				fmt.Printf("errror updating notification: %s\n", err)
				os.Exit(1)
			}
			fmt.Printf("notification (%d) updated\n", *notifNotIdFlag)
		case "d":
			utils.EnforceRequiredFlags(notifCmd, []string{"notId"})
			if err = notifications.DeleteNotification(*notifNotIdFlag); err != nil {
				fmt.Printf("errror deleting notification: %s\n", err)
				os.Exit(1)
			}
			fmt.Printf("notification (%d) deleted\n", *notifNotIdFlag)
		default:
			fmt.Printf("invalid action: %s\n", *notifActionFlag)
			os.Exit(1)
		}

	case "todo":
		exists, err := feat.FeatureExists(feat.E_todos_feature)
		if err != nil {
			fmt.Printf("error checking feature existence: %s\n", err)
			os.Exit(1)
		}
		if !exists {
			fmt.Println("feature \"todo\" not available")
			os.Exit(0)
		}
		if err = todoCmd.Parse(os.Args[2:]); err != nil {
			fmt.Printf("error parsing cli args: %s\n", err)
			os.Exit(1)
		}

		switch *todoActionFlag {
		case "c":
			utils.EnforceRequiredFlags(todoCmd, []string{"msgId"})
			if err := todos.CreateTodo(*todoMsgIdFlag); err != nil {
				fmt.Printf("error creating todo: %s\n", err)
				os.Exit(1)
			}
		case "r":
			if *todoTodIdFlag != -1 {
				if err = todos.ShowTodoDetails(*todoTodIdFlag); err != nil {
					fmt.Printf("error showing todos: %s\n", err)
					os.Exit(1)
				}
			} else {
				sort := "ASC"
				if *todoDescFlag == true {
					sort = "DESC"
				}

				if !slices.Contains([]string{"created_at", "updated_at", "status"}, strings.ToLower(*todoOrderFlag)) {
					fmt.Println("invalid value for '-order' flag")
					todoCmd.Usage()
					os.Exit(1)
				}
				if err = todos.ShowTodos(strings.ToLower(*todoOrderFlag), sort); err != nil {
					fmt.Printf("error showing todos: %s\n", err)
					os.Exit(1)
				}
			}
		case "u":
			utils.EnforceRequiredFlags(todoCmd, []string{"todId", "status"})
			if err = todos.UpdateTodo(*todoTodIdFlag, *todoStatusFlag); err != nil {
				fmt.Printf("error updating todo: %s\n", err)
				os.Exit(1)
			}
			fmt.Printf("todo (%d) updated\n", *todoTodIdFlag)
		case "d":
			utils.EnforceRequiredFlags(todoCmd, []string{"todId"})
			if err = todos.DeleteTodo(*todoTodIdFlag); err != nil {
				fmt.Printf("error deleting todo: %s\n", err)
				os.Exit(1)
			}
			fmt.Printf("todo (%d) deleted\n", *todoTodIdFlag)
		}
	default:
		fmt.Println("expected 'hello', 'show', 'create', 'update', 'delete' or 'notify' subcommand.")
		os.Exit(1)
	}
}
