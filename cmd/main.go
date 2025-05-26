package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/matheusbucater/gry/internal/db/sqlc"
	_ "modernc.org/sqlite"
)

var ddl string

type notificationTypeEnum int
const (
	simple notificationTypeEnum = iota
	recurring
)
var notificationTypeName = map[notificationTypeEnum]string{
	simple:    "simple",
	recurring: "recurring",
}
func (nte notificationTypeEnum) String() string {
	return notificationTypeName[nte]
}

type featureEnum int
const (
	notifications featureEnum = iota
)
var featureName = map[featureEnum]string{
	notifications: "notifications",
}
func (fe featureEnum) String() string {
	return featureName[fe]
}

func localizeDateTime(datetime time.Time) string {
	yearReplacer := strings.NewReplacer(
		"January", "Janeiro",
		"February", "Fevereiro",
		"March", "Março",
		"April", "Abril",
		"May", "Maio",
		"June", "Junho",
		"July", "Julho",
		"August", "Agosto",
		"September", "Setembro",
		"October", "Outubro",
		"November", "Novembro",
		"December", "Dezembro", 
	)
	dayReplacer := strings.NewReplacer(
		"Mon", "Seg",
		"Tue", "Ter",
		"Wed", "Qua",
		"Thu", "Qui",
		"Fri", "Sex",
		"Sat", "Sab",
		"Sun", "Dom",
	)

	return dayReplacer.Replace(yearReplacer.Replace(datetime.Format("Mon 02 Jan 2006 (15:04:05)")))
}

func enforceRequiredFlags(cmd *flag.FlagSet, required []string) {
	seen := make(map[string]bool)
	cmd.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	for _, req := range required {
		if !seen[req] {
			fmt.Printf("missing required '-%s' flag.\n", req)
			os.Exit(1)
		}
	}
}

func dbConnect(ctx context.Context) (*sql.DB, error) {
	if err := os.MkdirAll("./data", 0755); err != nil {
    	return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

    db, err := sql.Open("sqlite", "file:./data/messages.db?_foreign_keys=1&_journal_mode=WAL&mode=rwc")	
	if err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return nil, err
	}

	return db, nil
}

func parseWeekDays(wdString string) ([]time.Weekday, error) {
	wdAbbrev := []string{"su","mo","tu","we","th","fr","sa"}
	var parsedWD []time.Weekday

	if len(wdString) < 2 {
		return nil, errors.New("Invalid string " + "\"" + wdString + "\"" + ". Use \"su[sep=,]mo,tu,we,th,fr,sa\"")
	}
	if len(wdString) == 2 && !slices.Contains(wdAbbrev, wdString) {
			return nil, errors.New("Invalid string " + "\"" + wdString + "\"" + ". Use \"su[sep=,]mo,tu,we,th,fr,sa\"")
	}
	if len(wdString) > 2 && !strings.Contains(wdString, ",") {
		return nil, errors.New("Invalid string. Use \"su[sep=,]mo,tu,we,th,fr,sa\"")
	}

	for wd := range strings.SplitSeq(wdString, ",") {
		if !slices.Contains(wdAbbrev, wd) {
			return nil, errors.New("Invalid string " + "\"" + wd + "\"" + ". Use \"su[sep=,]mo,tu,we,th,fr,sa\"")
		}
		switch wd {
		case "su":
			parsedWD = append(parsedWD, time.Sunday)
		case "mo":
			parsedWD = append(parsedWD, time.Monday)
		case "tu":
			parsedWD = append(parsedWD, time.Tuesday)
		case "we":
			parsedWD = append(parsedWD, time.Wednesday)
		case "th":
			parsedWD = append(parsedWD, time.Thursday)
		case "fr":
			parsedWD = append(parsedWD, time.Friday)
		case "sa":
			parsedWD = append(parsedWD, time.Saturday)
		default:
			return nil, errors.New("Invalid string " + "\"" + wd + "\"" + ". Use \"su[sep=,]mo,tu,we,th,fr,sa\"")
		}
	}

	return parsedWD, nil
}

func showMessageDetails(id int64) error {
	ctx := context.Background()
	db, err := dbConnect(ctx)
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
		localizeDateTime(message.CreatedAt),
		localizeDateTime(message.UpdatedAt),
	)

	features, err := queries.GetFeaturesByMessageId(ctx, message.ID)
	if err != nil {
		return err
	}
	if len(features) != 0 {
		fmt.Printf("\n\tfeatures: %s", strings.Join(features, ","))
	}

	fmt.Println()

	return nil
}

func showMessages(order string, sort string) error {
	ctx := context.Background()
	db, err := dbConnect(ctx)
	if err != nil {
		return err
	}

	queries := sqlc.New(db)

	messages := []sqlc.Message{}
	switch order {
	case "created_at":
		if sort == "ASC" {
			messages, err = queries.GetMessagesOrderByCreatedAtASC(ctx);
		} else {
			messages, err = queries.GetMessagesOrderByCreatedAtDESC(ctx);
		}
	case "updated_at":
		if sort == "ASC" {
			messages, err = queries.GetMessagesOrderByUpdatedAtASC(ctx);
		} else {
			messages, err = queries.GetMessagesOrderByUpdatedAtDESC(ctx);
		}
	case "text":
		if sort == "ASC" {
			messages, err = queries.GetMessagesOrderByTextASC(ctx);
		} else {
			messages, err = queries.GetMessagesOrderByTextDESC(ctx);
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

		var sb strings.Builder
		for _, feat := range features {
			sb.WriteString(" [")
			sb.WriteString(strings.ToUpper(string(feat[0])))
			sb.WriteString("]")
		}
		if sb.Len() != 0 {
			fmt.Print(sb.String())
		}

		fmt.Println()
	}
	
	return nil
}

func showFeatures() error {
	ctx := context.Background()
	db, err := dbConnect(ctx)
	if err != nil {
		return err
	}

	queries := sqlc.New(db)

	features, err := queries.GetFeatures(ctx)
	if err != nil {
		return err
	}

	featuresCount := len(features)

	var sb strings.Builder
	sb.WriteString(strconv.Itoa(featuresCount))
	sb.WriteString(" feature")
	if featuresCount == 0 || featuresCount > 1 {
		sb.WriteString("s")
	} 
	sb.WriteString(" available")
	fmt.Println(sb.String())
	for _, feat := range features {
		fmt.Printf("\n[%s] %s\n", strings.ToUpper(string(feat.Name[0])), strings.ToLower(feat.Name))
	}
	
	return nil
}

func createMessage(message string) (sqlc.Message, error) {
	ctx := context.Background()
	db, err := dbConnect(ctx)
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
	db, err := dbConnect(ctx)
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
	db, err := dbConnect(ctx)
	if err != nil {
		return err
	}

	queries := sqlc.New(db)

	exists, err := queries.MessageExists(ctx, id)
	if (exists == 0) {
		return errors.New("Invalid message ID")
	}
	
	err = queries.DeleteMessage(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

func createSimpleNotification (msgId int64, triggerAt time.Time) error {
	ctx := context.Background()
	db, err := dbConnect(ctx)
	if err != nil {
		return err
	}

	queries := sqlc.New(db)

	exists, err := queries.MessageExists(ctx, msgId)
	if (exists == 0) {
		return errors.New("Invalid message ID")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	qtx := queries.WithTx(tx)

	notification, err := qtx.CreateNotification(ctx, sqlc.CreateNotificationParams{
		MessageID: msgId,
		Type: notificationTypeEnum.String(simple),
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	err = qtx.CreateSimpleNotification(ctx, sqlc.CreateSimpleNotificationParams{
		NotificationID: notification.ID,
		TriggerAt: triggerAt,
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	exists, err = qtx.MessageHasFeature(ctx, sqlc.MessageHasFeatureParams{
		MessageID: msgId,
		FeatureName: "notifications",
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	if exists == 1 {
		if err := tx.Commit(); err != nil {
			return err
		}

		return nil
	}

	err = qtx.CreateMessageFeature(ctx, sqlc.CreateMessageFeatureParams{
		MessageID: msgId,
		FeatureName: "notifications",
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func createRecurringNotification(msgId int64, weekDays []time.Weekday, triggerAt time.Time) error {
	ctx := context.Background()
	db, err := dbConnect(ctx)
	if err != nil {
		return err
	}

	queries := sqlc.New(db)

	exists, err := queries.MessageExists(ctx, msgId)
	if (exists == 0) {
		return errors.New("Invalid message ID")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	qtx := queries.WithTx(tx)

	notification, err := qtx.CreateNotification(ctx, sqlc.CreateNotificationParams{
		MessageID: msgId,
		Type: notificationTypeEnum.String(recurring),
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	recurring_notification, err := qtx.CreateRecurringNotification(ctx, sqlc.CreateRecurringNotificationParams{
		NotificationID: notification.ID,
		TriggerAtTime: sql.NullString{String: triggerAt.Format("15-04-05"), Valid: true},
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, wd := range weekDays {
		err = qtx.CreateRecurringNotificationDay(ctx, sqlc.CreateRecurringNotificationDayParams{
			RecurringNotificationID: recurring_notification.NotificationID,
			WeekDay: strings.ToLower(wd.String()),
		})
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	exists, err = qtx.MessageHasFeature(ctx, sqlc.MessageHasFeatureParams{
		MessageID: msgId,
		FeatureName: "notifications",
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	if exists == 1 {
		if err := tx.Commit(); err != nil {
			return err
		}
		return nil
	}

	err = qtx.CreateMessageFeature(ctx, sqlc.CreateMessageFeatureParams{
		MessageID: msgId,
		FeatureName: "notifications",
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func showNotifications() error {
	ctx := context.Background()
	db, err := dbConnect(ctx)
	if err != nil {
		return err
	}

	queries := sqlc.New(db)

	notifications, err := queries.GetNotifications(ctx)
	if err != nil {
		return err
	}
	
	notificationsCount := len(notifications)

	var sb strings.Builder
	sb.WriteString("You have ")
	sb.WriteString(strconv.Itoa(notificationsCount))
	sb.WriteString(" notification")
	
	if notificationsCount <= 0 {
		sb.WriteString("s")
	} else if notificationsCount == 1 {
		sb.WriteString("\n")
	} else {
		sb.WriteString("s\n")
	}
	fmt.Println(sb.String())

	for _, notification := range notifications {
		message, err := queries.GetMessageById(ctx, notification.MessageID)
		if err != nil {
			return err
		}

		var sb strings.Builder
		sb.WriteString("[")
		sb.WriteString(strings.ToUpper(string(notification.Type[0])))
		sb.WriteString("] ")
		sb.WriteString("\"")
		sb.WriteString(message.Text)
		sb.WriteString("\"")

		switch notification.Type {
		case notificationTypeEnum.String(simple):
			notification_details, err := queries.GetSimpleNotificationByNotificationId(ctx, notification.ID)
			if err != nil {
				return err
			}
			sb.WriteString(" at ")
			sb.WriteString(localizeDateTime(notification_details.TriggerAt))
		case notificationTypeEnum.String(recurring):
			notification_details, err := queries.GetRecurringNotificationByNotificationId(ctx, notification.ID)
			if err != nil {
				return err
			}
			sb.WriteString(" at ")
			sb.WriteString(strings.ReplaceAll(notification_details.TriggerAtTime.String, "-", ":"))
			sb.WriteString(" on ")

			notification_days, err := queries.GetRecurringNotificationDaysByNotificationId(ctx, notification.ID)
			if err != nil {
				return err
			}

			for i, nd := range notification_days {
				sb.WriteString(nd.WeekDay)
				sb.WriteString("s")
				if i == len(notification_days) - 2 { sb.WriteString(" and ") }
				if i < len(notification_days) - 2 { sb.WriteString(", ") } 
			}
		}

		fmt.Println(sb.String())
	}
	
	return nil
}

func notify() error {
	ctx := context.Background()
	db, err := dbConnect(ctx)
	if err != nil {
		return err
	}

	queries := sqlc.New(db)

	notifications, err := queries.GetNotifications(ctx)
	if err != nil {
		return err
	}

	if len(notifications) == 0 {
		return nil
	}
	
	fmt.Println()
	for _, notification := range notifications {
		switch notification.Type {
		case notificationTypeEnum.String(simple):
			notification_details, err := queries.GetSimpleNotificationByNotificationId(ctx, notification.ID)
			if err != nil {
				return err
			}
			if timeDiff := notification_details.TriggerAt.Sub(time.Now()); timeDiff <= 0{
				message, err := queries.GetMessageById(ctx, notification.MessageID)
				if err != nil {
					return err
				}
				fmt.Printf("[%s] \"%s\" (%s)\n", strings.ToUpper(string(notification.Type[0])), message.Text, timeDiff.Round(time.Second))
			}
		case notificationTypeEnum.String(recurring):
			notification_details, err := queries.GetRecurringNotificationByNotificationId(ctx, notification.ID)
			if err != nil {
				return err
			}
			exists, err := queries.RecurringNotificationHasDay(ctx, sqlc.RecurringNotificationHasDayParams{
				RecurringNotificationID: notification.ID,
				WeekDay: strings.ToLower(time.Now().Weekday().String()),
			})
			if err != nil {
				return err
			}
			if exists != 1 {
				return nil
			}

			triggerAt, err := time.Parse("15-04-05", notification_details.TriggerAtTime.String)
			now := time.Now()
			triggerAt = time.Date(
				now.Year(), now.Month(), now.Day(), 
				triggerAt.Hour(), triggerAt.Minute(), triggerAt.Second(),
				now.Nanosecond(), now.Location(),
			)
			if err != nil {
				return err
			}

			if timeDiff := triggerAt.Sub(time.Now()); timeDiff <= 0 {
				message, err := queries.GetMessageById(ctx, notification.MessageID)
				if err != nil {
					return err
				}
				fmt.Printf("[%s] \"%s\" (%s)\n", strings.ToUpper(string(notification.Type[0])), message.Text, timeDiff.Round(time.Second))
			}
		}
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
	showFeaturesFlag := showCmd.Bool("feat", false, "show all features available")
	showNotificationsFlag := showCmd.Bool("notif", false, "show all notifications")
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

	notifyCmd := flag.NewFlagSet("notify", flag.ExitOnError)
	notifyRecurringFlag := notifyCmd.Bool("recur", false, "use recurring notification type")
	notifyMsgIdFlag := notifyCmd.Int64("msgId", -1, "id of the message to be notified")
	notifyTriggerAtDFlag := notifyCmd.String("triggerAtD", "", "date to trigger the notification\nlayout: DD/MM/YY HH-MM-SS")
	notifyWeekDaysFlag := notifyCmd.String("weekDays", "", "week days that trigger the notification\n(su,mo,tu,we,th,fr,sa)")
	notifyTriggerAtTFlag := notifyCmd.String("triggerAtT", "", "time to trigger the notification\nlayout: HH-MM-SS")

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
		enforceRequiredFlags(helloCmd, []string{"name"})

		fmt.Printf("Hello %s!\n", *helloNameFlag)
		err = notify()
		if err != nil {
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
			err = showFeatures();
			if err != nil {
				fmt.Printf("error showing features: %s\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		}

		if *showNotificationsFlag == true {
			err = showNotifications();
			if err != nil {
				fmt.Printf("error showing notifications: %s\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		}

		if *showIdFlag != -1 {
			err = showMessageDetails(*showIdFlag)
			if err != nil {
				fmt.Printf("error showing message details: %s\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		}

		sort := "ASC"
		if *showDescFlag == true {
			sort = "DESC"
		}

		if !slices.Contains([]string{"created_at", "updated_at", "text"}, strings.ToLower(*showOrderFlag)) {
			fmt.Println("invalid value for '-order' flag")
			showCmd.Usage()
			os.Exit(1)
		}
		err = showMessages(strings.ToLower(*showOrderFlag), sort);
		if err != nil {
			fmt.Printf("error displaying messages: %s\n", err)
			os.Exit(1)
		}
	case "create":
		err := createCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("error parsing cli args: %s\n", err)
		}
		enforceRequiredFlags(createCmd, []string{"message"})
		newMessage, err := createMessage(*createMessageFlag)
		if err != nil {
			fmt.Printf("error creating new message: %s\n", err)
		}
		fmt.Println(newMessage)
	case "update":
		err := updateCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("error parsing cli args: %s\n", err)
			os.Exit(1)
		}
		enforceRequiredFlags(updateCmd, []string{"id", "message"})
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
		enforceRequiredFlags(deleteCmd, []string{"id"})
		err = deleteMessage(*deleteIdFlag)
		if err != nil {
			fmt.Printf("error deleting message: %s\n", err)
			os.Exit(1)
		}
		fmt.Println("Message deleted.")
	case "notify":
		err := notifyCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("error parsing cli args: %s\n", err)
			os.Exit(1)
		}
		if *notifyRecurringFlag {
			enforceRequiredFlags(notifyCmd, []string{"msgId", "weekDays", "triggerAtT"})
			weekDays, err := parseWeekDays(*notifyWeekDaysFlag)
			if err != nil {
				fmt.Printf("errror parsing weekDays: %s\n", err)
			}
			triggerAt, err := time.Parse(triggerAtTLayout, *notifyTriggerAtTFlag)
			if err != nil {
				fmt.Printf("errror parsing triggerAtT date: %s\n", err)
			}
			fmt.Println(triggerAt.String())
			fmt.Println(weekDays)
			err = createRecurringNotification(*notifyMsgIdFlag, weekDays, triggerAt)
			if err != nil {
				fmt.Printf("error creating notification: %s\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		} else {
			enforceRequiredFlags(notifyCmd, []string{"msgId", "triggerAtD"})
			triggerAt, err := time.Parse(triggerAtDLayout, *notifyTriggerAtDFlag)
			triggerAt = time.Date(
				triggerAt.Year(), triggerAt.Month(), triggerAt.Day(), 
				triggerAt.Hour(), triggerAt.Minute(), triggerAt.Second(),
				0, time.Local,
			)
			err = createSimpleNotification(*notifyMsgIdFlag, triggerAt)
			if err != nil {
				fmt.Printf("error creating notification: %s\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
		// if *notifyRecurringFlag {
		// 	enforceRequiredFlags(notifyCmd, []string{"msgId"})
		// 	err = createRecurringNotification(*notifyMsgIdFlag)
		// 	if err != nil {
		// 		fmt.Printf("error creating notification: %s\n", err)
		// 		os.Exit(1)
		// 	}
		// 	os.Exit(0)
		// }
	default:
		fmt.Println("expected 'hello', 'show', 'create', 'update', 'delete' or 'notify' subcommand.")
		os.Exit(1)
	}
}
