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

//============================================================================== 
// FEATURES
//------------------------------------------------------------------------------
type featureEnum int
const (
	e_notifications_feature featureEnum = iota
	e_todos_feature 
	e_feature_not_available
)
var featureName = map[featureEnum]string{
	e_notifications_feature: "notifications",
	e_todos_feature: "todos",
	e_feature_not_available: "not_available",
}
func (fe featureEnum) String() string {
	return featureName[fe]
}
//------------------------------------------------------------------------------
// Notifications
//------------------------------------------------------------------------------
type notificationTypeEnum int
const (
	e_simple_notification notificationTypeEnum = iota
	e_recurring_notification
)
var notificationTypeName = map[notificationTypeEnum]string{
	e_simple_notification:    "simple",
	e_recurring_notification: "recurring",
}
func (nte notificationTypeEnum) String() string {
	return notificationTypeName[nte]
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
		case notificationTypeEnum.String(e_simple_notification):
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
		case notificationTypeEnum.String(e_recurring_notification):
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
		sb.WriteString("(")
		sb.WriteString(fmt.Sprintf("%d", notification.ID))
		sb.WriteString(") ")
		sb.WriteString("\"")
		sb.WriteString(message.Text)
		sb.WriteString("\"")

		switch notification.Type {
		case notificationTypeEnum.String(e_simple_notification):
			notification_details, err := queries.GetSimpleNotificationByNotificationId(ctx, notification.ID)
			if err != nil {
				return err
			}
			sb.WriteString(" at ")
			sb.WriteString(localizeDateTime(notification_details.TriggerAt))
		case notificationTypeEnum.String(e_recurring_notification):
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

		sb.WriteString(" [")
		sb.WriteString(notification.Type[:5])
		sb.WriteString("]")

		fmt.Println(sb.String())
	}
	
	return nil
}

func showNotificationDetails(notId int64) error {
	ctx := context.Background()
	db, err := dbConnect(ctx)
	if err != nil {
		return err
	}

	queries := sqlc.New(db)

	exists, err := queries.NotificationExists(ctx, notId)
	if exists != 1 {
		return errors.New("notification does not exist")
	}
	notification, err := queries.GetNotificationById(ctx, notId)
	if err != nil {
		return err
	}

	var sb strings.Builder
	switch notification.Type {
	case e_simple_notification.String():
		notification_details, err := queries.GetSimpleNotificationByNotificationId(ctx, notification.ID)
		if err != nil {
			return err
		}
		sb.WriteString("\ttrigger_at: ")
		sb.WriteString(localizeDateTime(notification_details.TriggerAt))
	case e_recurring_notification.String():
	}

	fmt.Println("Notification details:")
	fmt.Printf("\tid: %d\n", notification.ID)
	fmt.Printf("\tmessage_id: %d\n", notification.MessageID)
	fmt.Printf("\ttype: %s\n", notification.Type)
	fmt.Println(sb.String())
	fmt.Printf("\tcreated_at: %s\n", localizeDateTime(notification.CreatedAt))
	fmt.Printf("\tupdated_at: %s\n", localizeDateTime(notification.UpdatedAt))

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
		Type: notificationTypeEnum.String(e_simple_notification),
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	if err = qtx.CreateSimpleNotification(ctx, sqlc.CreateSimpleNotificationParams{
		NotificationID: notification.ID,
		TriggerAt: triggerAt,
	}); err != nil {
		tx.Rollback()
		return err
	}

	exists, err = qtx.MessageHasFeature(ctx, sqlc.MessageHasFeatureParams{
		MessageID: msgId,
		FeatureName: e_notifications_feature.String(),
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	if exists == 1 {
		if err := qtx.IncrementMessageFeatureCount(ctx, sqlc.IncrementMessageFeatureCountParams{
			MessageID: msgId,
			FeatureName: e_notifications_feature.String(),
		}); err != nil { return err }

		if err := tx.Commit(); err != nil { return err }

		return nil
	}

	if err = qtx.CreateMessageFeature(ctx, sqlc.CreateMessageFeatureParams{
		MessageID: msgId,
		FeatureName: e_notifications_feature.String(),
	}); err != nil {
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
		Type: notificationTypeEnum.String(e_recurring_notification),
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
		if err = qtx.CreateRecurringNotificationDay(ctx, sqlc.CreateRecurringNotificationDayParams{
			RecurringNotificationID: recurring_notification.NotificationID,
			WeekDay: strings.ToLower(wd.String()),
		}); err != nil {
			tx.Rollback()
			return err
		}
	}

	exists, err = qtx.MessageHasFeature(ctx, sqlc.MessageHasFeatureParams{
		MessageID: msgId,
		FeatureName: e_notifications_feature.String(),
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	if exists == 1 {
		if err := qtx.IncrementMessageFeatureCount(ctx, sqlc.IncrementMessageFeatureCountParams{
			MessageID: msgId,
			FeatureName: e_notifications_feature.String(),
		}); err != nil { return err }

		if err := tx.Commit(); err != nil { return err }

		return nil
	}

	if err = qtx.CreateMessageFeature(ctx, sqlc.CreateMessageFeatureParams{
		MessageID: msgId,
		FeatureName: e_notifications_feature.String(),
	}); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func updateNotification(notId int64, triggerAt string, weekDays string) error {
	triggerAtDLayout := "02/01/06 15-04-05" // "DD/MM/YY HH-MM-SS"
	triggerAtTLayout := "15-04-05" // "HH-MM-SS"

	ctx := context.Background()
	db, err := dbConnect(ctx)
	if err != nil {
		return err
	}

	queries := sqlc.New(db)

	exists, err := queries.NotificationExists(ctx, notId)
	if err != nil { return err }
	if exists != 1 { return errors.New("notification does not exist") }

	notification, err := queries.GetNotificationById(ctx, notId)
	if err != nil { return err }
	
	switch notification.Type {
	case e_simple_notification.String():
		if weekDays != "" { return errors.New("Invalid flag -weekDays for simple notifications.") }

		triggerAt, err := time.Parse(triggerAtDLayout, triggerAt)
		if err != nil { return err }
		triggerAt = time.Date(
			triggerAt.Year(), triggerAt.Month(), triggerAt.Day(), 
			triggerAt.Hour(), triggerAt.Minute(), triggerAt.Second(),
			0, time.Local,
		)

		if _, err := queries.UpdateSimpleNotification(ctx, sqlc.UpdateSimpleNotificationParams{
			NotificationID: notId,
			TriggerAt: triggerAt,
		}); err != nil { return err }
	case e_recurring_notification.String():
		// TODO: allow user to update only the weekdays without having to pass the triggerAt time
		triggerAt, err := time.Parse(triggerAtTLayout, triggerAt)
		if err != nil { return err }

		if weekDays == "" {
			if _, err = queries.UpdateRecurringNotification(ctx, sqlc.UpdateRecurringNotificationParams{
				NotificationID: notId,
				TriggerAtTime: sql.NullString{String: triggerAt.Format("15-04-05"), Valid: true},
			}); err != nil {
				return err
			}
		} else {
			weekDays, err := parseWeekDays(weekDays)
			if err != nil { return err }

			for _, wd := range []time.Weekday{
				time.Sunday, time.Monday, time.Tuesday, time.Wednesday,
				time.Thursday, time.Friday, time.Saturday,
			} {
				exists, err := queries.RecurringNotificationHasDay(ctx, sqlc.RecurringNotificationHasDayParams{
					RecurringNotificationID: notId,
					WeekDay: strings.ToLower(wd.String()),
				})
				if err != nil { return err }

				if exists == 1 && !slices.Contains(weekDays, wd) {
					if err = queries.DeleteRecurringNotificationDayByNotificationId(ctx, sqlc.DeleteRecurringNotificationDayByNotificationIdParams{
						RecurringNotificationID: notId,
						WeekDay: strings.ToLower(wd.String()),
					}); err != nil { return err }
					continue
				}
				if exists != 1 && slices.Contains(weekDays, wd) {
					if err = queries.CreateRecurringNotificationDay(ctx, sqlc.CreateRecurringNotificationDayParams{
						RecurringNotificationID: notId,
						WeekDay: strings.ToLower(wd.String()),
					}); err != nil { return err }
					continue
				}
			}
		}
	}

	return nil
}

func deleteNotification(notId int64) error {
	ctx := context.Background()
	db, err := dbConnect(ctx)
	if err != nil {
		return err
	}

	queries := sqlc.New(db)

	exists, err := queries.NotificationExists(ctx, notId)
	if err != nil { return err }
	if exists != 1 { return errors.New("notification does not exist") }

	msgId, err := queries.DeleteNotificationByIdReturningMsgId(ctx, notId)
	if err = queries.DecrementMessageFeatureCount(ctx, sqlc.DecrementMessageFeatureCountParams{
		MessageID: msgId,
		FeatureName: e_notifications_feature.String(),
	}); err != nil {
		return err
	}

	return nil
}
//------------------------------------------------------------------------------
// Todos
//------------------------------------------------------------------------------
type todoStatusEnum int
const (
	e_pending_status todoStatusEnum = iota
	e_done_status
)
var todoStatusName = map[todoStatusEnum]string{
	e_pending_status: "pending",
	e_done_status: 	  "done",
}
func (nte todoStatusEnum) String() string {
	return todoStatusName[nte]
}

func showTodos() error {
	ctx := context.Background()
	db, err := dbConnect(ctx)
	if err != nil { return err }

	queries := sqlc.New(db)

	todos, err := queries.GetTodos(ctx)
	if err != nil { return err }

	todosCount := len(todos)

	var sb strings.Builder
	sb.WriteString("You have ")
	sb.WriteString(strconv.Itoa(todosCount))
	sb.WriteString(" todo")
	
	if todosCount <= 0 {
		sb.WriteString("s")
	} else if todosCount == 1 {
		sb.WriteString("\n")
	} else {
		sb.WriteString("s\n")
	}
	fmt.Println(sb.String())

	sb.Reset()
	for _, todo := range todos {
		sb.WriteString("(")
		sb.WriteString(fmt.Sprintf("%d", todo.ID))
		sb.WriteString(") ")

		sb.WriteString("\"")
		message, err := queries.GetMessageById(ctx, todo.MessageID)
		if err != nil { return err }
		sb.WriteString(message.Text)
		sb.WriteString("\" ")

		sb.WriteString(todo.Status)
		sb.WriteString("\n")
	}
	fmt.Print(sb.String())
	return nil
}

func showTodoDetails(todId int64) error {
	ctx := context.Background()
	db, err := dbConnect(ctx)
	if err != nil {
		return err
	}

	queries := sqlc.New(db)

	exists, err := queries.TodoExists(ctx, todId)
	if err != nil { return err }
	if exists != 1 { return errors.New("todo does not exist") }

	todo, err := queries.GetTodoById(ctx, todId)
	if err != nil { return err }

	fmt.Println("Todo details:")
	fmt.Printf("\tid: %d\n", todo.ID)
	fmt.Printf("\tmessage_id: %d\n", todo.MessageID)
	fmt.Printf("\tstatus: %s\n", todo.Status)
	fmt.Printf("\tcreated_at: %s\n", localizeDateTime(todo.CreatedAt))
	fmt.Printf("\tupdated_at: %s\n", localizeDateTime(todo.UpdatedAt))

	return nil
}

func createTodo(msgId int64) error {
	ctx := context.Background()
	db, err := dbConnect(ctx)
	if err != nil { return err }

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

	if _, err := qtx.CreateTodo(ctx, msgId); err != nil {
		tx.Rollback()
		return err
	}

	exists, err = qtx.MessageHasFeature(ctx, sqlc.MessageHasFeatureParams{
		MessageID: msgId,
		FeatureName: e_todos_feature.String(),
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	if exists == 1 {
		if err := qtx.IncrementMessageFeatureCount(ctx, sqlc.IncrementMessageFeatureCountParams{
			MessageID: msgId,
			FeatureName: e_todos_feature.String(),
		}); err != nil { return err }

		if err := tx.Commit(); err != nil { return err }

		return nil
	}

	if err = qtx.CreateMessageFeature(ctx, sqlc.CreateMessageFeatureParams{
		MessageID: msgId,
		FeatureName: e_todos_feature.String(),
	}); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func updateTodo(todId int64, status string) error {
	ctx := context.Background()
	db, err := dbConnect(ctx)
	if err != nil { return err }

	queries := sqlc.New(db)

	exists, err := queries.TodoExists(ctx, todId)
	if err != nil { return err }
	if (exists == 0) { return errors.New("Invalid todo ID") }

	switch status {
	case e_pending_status.String():
		if _, err := queries.UpdateTodoById(ctx, sqlc.UpdateTodoByIdParams{
			ID: todId,
			Status: e_pending_status.String(),
		}); err != nil { return err }
	case e_done_status.String():
		if _, err := queries.UpdateTodoById(ctx, sqlc.UpdateTodoByIdParams{
			ID: todId,
			Status: e_done_status.String(),
		}); err != nil { return err }
	default:
		return fmt.Errorf("Invalid -status \"%s\"", status)
	}


	return nil
}

func deleteTodo(todId int64) error {
	ctx := context.Background()
	db, err := dbConnect(ctx)
	if err != nil { return err }

	queries := sqlc.New(db)

	exists, err := queries.TodoExists(ctx, todId)
	if err != nil { return err }
	if (exists == 0) { return errors.New("Invalid todo ID") }

	msgId, err := queries.DeleteTodoByIdReturningMsgId(ctx, todId)
	if err = queries.DecrementMessageFeatureCount(ctx, sqlc.DecrementMessageFeatureCountParams{
		MessageID: msgId,
		FeatureName: e_todos_feature.String(),
	}); err != nil { return err }

	if err = queries.DeleteTodoById(ctx, todId); err != nil { return err }

	return nil
}
//============================================================================== 

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

func featureExists(feat featureEnum) (bool, error) {
	ctx := context.Background()
	db, err := dbConnect(ctx)
	if err != nil {
		return false, err
	}
	queries := sqlc.New(db)

	exists, err := queries.FeatureExists(ctx, feat.String())
	return exists == 1, nil
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
	if featuresCount > 0 {
		sb.WriteString("\n")
	}
	fmt.Println(sb.String())
	for _, feat := range features {
		fmt.Printf("%s\n", strings.ToLower(feat.Name))
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

	todoCmd := flag.NewFlagSet("todo", flag.ExitOnError)
	todoActionFlag := todoCmd.String("a", "r", "action:\n\t\"c\" create,\n\t\"r\" read,\n\t\"u\" update,\n\t\"d\" delete")
	todoMsgIdFlag := todoCmd.Int64("msgId", -1, "message id")
	todoTodIdFlag := todoCmd.Int64("todId", -1, "todo id")
	todoStatusFlag := todoCmd.String("status", "", "todo status\n(pending,done)")

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
		if err = notify(); err != nil {
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
			if err = showFeatures(); err != nil {
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
			if err = showMessages(strings.ToLower(*showOrderFlag), sort); err != nil {
				fmt.Printf("error displaying messages: %s\n", err)
				os.Exit(1)
			}
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
		fmt.Printf("message created: (%d) \"%s\".\n", newMessage.ID, newMessage.Text)
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
		if err = deleteMessage(*deleteIdFlag); err != nil {
			fmt.Printf("error deleting message: %s\n", err)
			os.Exit(1)
		}
		fmt.Println("Message deleted.")
	case "notif":
		exists, err := featureExists(e_notifications_feature)
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
				enforceRequiredFlags(notifCmd, []string{"msgId", "weekDays", "triggerAt"})
				weekDays, err := parseWeekDays(*notifWeekDaysFlag)
				if err != nil {
					fmt.Printf("errror parsing weekDays: %s\n", err)
					os.Exit(1)
				}
				triggerAt, err := time.Parse(triggerAtTLayout, *notifTriggerAtFlag)
				if err != nil {
					fmt.Printf("errror parsing triggerAtT date: %s\n", err)
					os.Exit(1)
				}
				if err = createRecurringNotification(*notifMsgIdFlag, weekDays, triggerAt); err != nil {
					fmt.Printf("error creating notification: %s\n", err)
					os.Exit(1)
				}
				os.Exit(0)
			} else {
				enforceRequiredFlags(notifCmd, []string{"msgId", "triggerAt"})
				triggerAt, err := time.Parse(triggerAtDLayout, *notifTriggerAtFlag)
				triggerAt = time.Date(
					triggerAt.Year(), triggerAt.Month(), triggerAt.Day(), 
					triggerAt.Hour(), triggerAt.Minute(), triggerAt.Second(),
					0, time.Local,
				)
				if err = createSimpleNotification(*notifMsgIdFlag, triggerAt); err != nil {
					fmt.Printf("error creating notification: %s\n", err)
					os.Exit(1)
				}
				os.Exit(0)
			}
		case "r":
			if *notifNotIdFlag != -1 {
				if err = showNotificationDetails(*notifNotIdFlag); err != nil {
					fmt.Printf("error showing notification details: %s\n", err)
					os.Exit(1)
				}
			} else {
				if err = showNotifications(); err != nil {
					fmt.Printf("error showing notifications: %s\n", err)
					os.Exit(1)
				}
			}
		case "u":
			enforceRequiredFlags(notifCmd, []string{"notId", "triggerAt"})
			triggerAt, err := time.Parse(triggerAtDLayout, *notifTriggerAtFlag)
			triggerAt = time.Date(
				triggerAt.Year(), triggerAt.Month(), triggerAt.Day(), 
				triggerAt.Hour(), triggerAt.Minute(), triggerAt.Second(),
				0, time.Local,
			)
			if err = updateNotification(*notifNotIdFlag, *notifTriggerAtFlag, *notifWeekDaysFlag); err != nil {
				fmt.Printf("errror updating notification: %s\n", err)
				os.Exit(1)
			}
			fmt.Printf("notification (%d) updated\n", *notifNotIdFlag)
		case "d":
			enforceRequiredFlags(notifCmd, []string{"notId"})
			if err = deleteNotification(*notifNotIdFlag); err != nil {
				fmt.Printf("errror deleting notification: %s\n", err)
				os.Exit(1)
			}
			fmt.Printf("notification (%d) deleted\n", *notifNotIdFlag)
		default:
			fmt.Printf("invalid action: %s\n", *notifActionFlag)
			os.Exit(1)
		}

	case "todo":
		exists, err := featureExists(e_todos_feature)
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
			enforceRequiredFlags(todoCmd, []string{"msgId"})
			if err := createTodo(*todoMsgIdFlag); err != nil {
				fmt.Printf("error creating todo: %s\n", err)
				os.Exit(1)
			}
		case "r":
			if *todoTodIdFlag != -1 {
				if err = showTodoDetails(*todoTodIdFlag); err != nil {
					fmt.Printf("error showing todos: %s\n", err)
					os.Exit(1)
				}
			} else {
				if err = showTodos(); err != nil {
					fmt.Printf("error showing todos: %s\n", err)
					os.Exit(1)
				}
			}
		case "u":
			enforceRequiredFlags(todoCmd, []string{"todId", "status"})
			if err = updateTodo(*todoTodIdFlag, *todoStatusFlag); err != nil {
				fmt.Printf("error updating todo: %s\n", err)
				os.Exit(1)
			}
			fmt.Printf("todo (%d) updated\n", *todoTodIdFlag)
		case "d":
			enforceRequiredFlags(todoCmd, []string{"todId"})
			if err = deleteTodo(*todoTodIdFlag); err != nil {
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
