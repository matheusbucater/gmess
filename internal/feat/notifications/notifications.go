package notifications

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

	"github.com/matheusbucater/gmess/internal/db/sqlc"
	// "github.com/matheusbucater/gmess/internal/feat"
	"github.com/matheusbucater/gmess/internal/utils"
)

type notificationTypeEnum int
const (
	e_simple_notification notificationTypeEnum = iota
	e_recurring_notification
)
var notificationTypeName = map[notificationTypeEnum]string{
	e_simple_notification:    "simple",
	e_recurring_notification: "recurring",
}
func (nte notificationTypeEnum) string() string {
	return notificationTypeName[nte]
}

func notify() error {
	ctx := context.Background()
	db, err := utils.DbConnect(ctx)
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
		case notificationTypeEnum.string(e_simple_notification):
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
		case notificationTypeEnum.string(e_recurring_notification):
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

func showNotification(order string, sort string) error {
	ctx := context.Background()
	db, err := utils.DbConnect(ctx)
	if err != nil { return err }

	queries := sqlc.New(db)

	notifications := []sqlc.Notification{}
	switch order {
	case "created_at":
		if sort == "ASC" {
			notifications, err = queries.GetNotificationsOrderByCreatedAtASC(ctx)
		} else {
			notifications, err = queries.GetNotificationsOrderByCreatedAtDESC(ctx)
		}
	case "updated_at":
		if sort == "ASC" {
			notifications, err = queries.GetNotificationsOrderByUpdatedAtASC(ctx)
		} else {
			notifications, err = queries.GetNotificationsOrderByUpdatedAtDESC(ctx)
		}
	case "type":
		if sort == "ASC" {
			notifications, err = queries.GetNotificationsOrderByTypeASC(ctx)
		} else {
			notifications, err = queries.GetNotificationsOrderByTypeDESC(ctx)
		}
	}
	if err != nil { return err }
	
	notificationsCount := len(notifications)
	if notificationsCount> 0 {
		fmt.Printf("(order by: '%s' %s)\n\n", order, sort)
	}

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
		if err != nil { return err }

		var sb strings.Builder
		sb.WriteString("(")
		sb.WriteString(fmt.Sprintf("%d", notification.ID))
		sb.WriteString(") ")
		sb.WriteString("\"")
		sb.WriteString(message.Text)
		sb.WriteString("\"")

		switch notification.Type {
		case notificationTypeEnum.string(e_simple_notification):
			notification_details, err := queries.GetSimpleNotificationByNotificationId(ctx, notification.ID)
			if err != nil {	return err }

			sb.WriteString(" at ")
			sb.WriteString(utils.LocalizeDateTime(notification_details.TriggerAt))
		case notificationTypeEnum.string(e_recurring_notification):
			notification_details, err := queries.GetRecurringNotificationByNotificationId(ctx, notification.ID)
			if err != nil { return err }

			sb.WriteString(" at ")
			sb.WriteString(strings.ReplaceAll(notification_details.TriggerAtTime.String, "-", ":"))
			sb.WriteString(" on ")

			notification_days, err := queries.GetRecurringNotificationDaysByNotificationId(ctx, notification.ID)
			if err != nil { return err }

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
	db, err := utils.DbConnect(ctx)
	if err != nil { return err }

	queries := sqlc.New(db)

	exists, err := queries.NotificationExists(ctx, notId)
	if err != nil { return err }
	if exists != 1 { return errors.New("notification does not exist") }

	notificationAndMessage, err := queries.GetNotificationAndMessageById(ctx, notId)
	if err != nil { return err }

	notification := notificationAndMessage.Notification
	message := notificationAndMessage.Message

	var sb strings.Builder
	switch notification.Type {
	case e_simple_notification.string():
		notification_details, err := queries.GetSimpleNotificationByNotificationId(ctx, notification.ID)
		if err != nil { return err }

		sb.WriteString("\t  trigger_at: ")
		sb.WriteString(utils.LocalizeDateTime(notification_details.TriggerAt))
	case e_recurring_notification.string():
		notification_details, err := queries.GetRecurringNotificationByNotificationId(ctx, notification.ID)
		if err != nil { return err }

		sb.WriteString("\t  trigger_at_time: ")
		sb.WriteString(strings.ReplaceAll(notification_details.TriggerAtTime.String, "-", ":"))
		sb.WriteString("\n")

		notification_days, err := queries.GetRecurringNotificationDaysByNotificationId(ctx, notification.ID)
		if err != nil { return err }

		sb.WriteString("\t  week_days: ")
		for i, nd := range notification_days {
			sb.WriteString(nd.WeekDay)
			sb.WriteString("s")
			if i == len(notification_days) - 2 { sb.WriteString(" and ") }
			if i < len(notification_days) - 2 { sb.WriteString(", ") } 
		}
	}

	fmt.Println("Notification details:")
	fmt.Printf("\tid: %d\n", notification.ID)
	fmt.Printf("\tmessage:\n")
	fmt.Printf("\t  id: %d\n", message.ID)
	fmt.Printf("\t  text: %s\n", message.Text)
	fmt.Printf("\t  created_at: %s\n", utils.LocalizeDateTime(message.CreatedAt))
	fmt.Printf("\t  updated_at: %s\n", utils.LocalizeDateTime(message.UpdatedAt))
	fmt.Printf("\ttype: %s\n", notification.Type)
	fmt.Println(sb.String())
	fmt.Printf("\tcreated_at: %s\n", utils.LocalizeDateTime(notification.CreatedAt))
	fmt.Printf("\tupdated_at: %s\n", utils.LocalizeDateTime(notification.UpdatedAt))

	return nil
}

func createSimpleNotification (msgId int64, triggerAt time.Time) error {
	ctx := context.Background()
	db, err := utils.DbConnect(ctx)
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
		Type: notificationTypeEnum.string(e_simple_notification),
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
		// FeatureName: feat.E_notifications_feature.String(), 
		FeatureName: "notification", // TODO: fix this mess
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	if exists == 1 {
		if err := qtx.IncrementMessageFeatureCount(ctx, sqlc.IncrementMessageFeatureCountParams{
			MessageID: msgId,
			// FeatureName: feat.E_notifications_feature.String(), 
			FeatureName: "notification", // TODO: fix this mess
		}); err != nil { return err }

		if err := tx.Commit(); err != nil { return err }

		return nil
	}

	if err = qtx.CreateMessageFeature(ctx, sqlc.CreateMessageFeatureParams{
		MessageID: msgId,
		// FeatureName: feat.E_notifications_feature.String(), 
		FeatureName: "notification", // TODO: fix this mess
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
	db, err := utils.DbConnect(ctx)
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
		Type: notificationTypeEnum.string(e_recurring_notification),
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
		// FeatureName: feat.E_notifications_feature.String(), 
		FeatureName: "notification", // TODO: fix this mess
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	if exists == 1 {
		if err := qtx.IncrementMessageFeatureCount(ctx, sqlc.IncrementMessageFeatureCountParams{
			MessageID: msgId,
			// FeatureName: feat.E_notifications_feature.String(), 
			FeatureName: "notification", // TODO: fix this mess
		}); err != nil { return err }

		if err := tx.Commit(); err != nil { return err }

		return nil
	}

	if err = qtx.CreateMessageFeature(ctx, sqlc.CreateMessageFeatureParams{
		MessageID: msgId,
		// FeatureName: feat.E_notifications_feature.String(), 
		FeatureName: "notification", // TODO: fix this mess
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
	db, err := utils.DbConnect(ctx)
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
	case e_simple_notification.string():
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
	case e_recurring_notification.string():
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
			weekDays, err := utils.ParseWeekDays(weekDays)
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
	db, err := utils.DbConnect(ctx)
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
		// FeatureName: feat.E_notifications_feature.String(), 
		FeatureName: "notification", // TODO: fix this mess
	}); err != nil {
		return err
	}

	return nil
}

func Cmd(args []string) {
	triggerAtDLayout := "02/01/06 15-04-05" // "DD/MM/YY HH-MM-SS"
	triggerAtTLayout := "15-04-05" // "HH-MM-SS"

	cmd := flag.NewFlagSet("notif", flag.ExitOnError)
	actionFlag := cmd.String("a", "r", "action:\n\t\"c\" create,\n\t\"r\" read,\n\t\"u\" update,\n\t\"d\" delete")
	recurringFlag := cmd.Bool("recur", false, "use recurring notification type")
	msgIdFlag := cmd.Int64("msgId", -1, "message id")
	triggerAtFlag := cmd.String("triggerAt", "", "trigger notification at\nlayout: DD/MM/YY HH-MM-SS or HH-MM-SS")
	weekDaysFlag := cmd.String("weekDays", "", "week days that trigger the notification\n(su,mo,tu,we,th,fr,sa)")
	notIdFlag := cmd.Int64("notId", -1, "notification id")
	// TODO: add support to order by trigger_at
	orderFlag := cmd.String("order", "created_at", "order by: 'created_at', 'updated_at' or 'type'")
	descFlag := cmd.Bool("desc", false, "retrieve notifications in descending order")

	if err := cmd.Parse(args); err != nil {
		fmt.Printf("error parsing cli args: %s\n", err)
		os.Exit(1)
	}

	switch *actionFlag {
	case "c":
		if *recurringFlag == true {
			utils.EnforceRequiredFlags(cmd, []string{"msgId", "weekDays", "triggerAt"})
			weekDays, err := utils.ParseWeekDays(*weekDaysFlag)
			if err != nil {
				fmt.Printf("errror parsing weekDays: %s\n", err)
				os.Exit(1)
			}
			triggerAt, err := time.Parse(triggerAtTLayout, *triggerAtFlag)
			if err != nil {
				fmt.Printf("errror parsing triggerAtT date: %s\n", err)
				os.Exit(1)
			}
			if err = createRecurringNotification(*msgIdFlag, weekDays, triggerAt); err != nil {
				fmt.Printf("error creating notification: %s\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		} else {
			utils.EnforceRequiredFlags(cmd, []string{"msgId", "triggerAt"})
			triggerAt, err := time.Parse(triggerAtDLayout, *triggerAtFlag)
			triggerAt = time.Date(
				triggerAt.Year(), triggerAt.Month(), triggerAt.Day(), 
				triggerAt.Hour(), triggerAt.Minute(), triggerAt.Second(),
				0, time.Local,
			)
			if err = createSimpleNotification(*msgIdFlag, triggerAt); err != nil {
				fmt.Printf("error creating notification: %s\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	case "r":
		if *notIdFlag != -1 {
			if err := showNotificationDetails(*notIdFlag); err != nil {
				fmt.Printf("error showing notification details: %s\n", err)
				os.Exit(1)
			}
		} else {
			sort := "ASC"
			if *descFlag == true {
				sort = "DESC"
			}

			if !slices.Contains([]string{"created_at", "updated_at", "type"}, strings.ToLower(*orderFlag)) {
				fmt.Println("invalid value for '-order' flag")
				cmd.Usage()
				os.Exit(1)
			}
			if err := showNotification(strings.ToLower(*orderFlag), sort); err != nil {
				fmt.Printf("error showing notifications: %s\n", err)
				os.Exit(1)
			}
		}
	case "u":
		utils.EnforceRequiredFlags(cmd, []string{"notId", "triggerAt"})
		triggerAt, err := time.Parse(triggerAtDLayout, *triggerAtFlag)
		triggerAt = time.Date(
			triggerAt.Year(), triggerAt.Month(), triggerAt.Day(), 
			triggerAt.Hour(), triggerAt.Minute(), triggerAt.Second(),
			0, time.Local,
		)
		if err = updateNotification(*notIdFlag, *triggerAtFlag, *weekDaysFlag); err != nil {
			fmt.Printf("errror updating notification: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("notification (%d) updated\n", *notIdFlag)
	case "d":
		utils.EnforceRequiredFlags(cmd, []string{"notId"})
		if err := deleteNotification(*notIdFlag); err != nil {
			fmt.Printf("errror deleting notification: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("notification (%d) deleted\n", *notIdFlag)
	default:
		fmt.Printf("invalid action: %s\n", *actionFlag)
		os.Exit(1)
	}
}
