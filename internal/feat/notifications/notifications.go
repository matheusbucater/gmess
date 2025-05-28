package notifications

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"strconv"

	"github.com/matheusbucater/gmess/internal/feat"
	"github.com/matheusbucater/gmess/internal/db/sqlc"
	"github.com/matheusbucater/gmess/internal/utils"
)

type NotificationTypeEnum int
const (
	E_simple_notification NotificationTypeEnum = iota
	e_recurring_notification
)
var notificationTypeName = map[NotificationTypeEnum]string{
	E_simple_notification:    "simple",
	e_recurring_notification: "recurring",
}
func (nte NotificationTypeEnum) String() string {
	return notificationTypeName[nte]
}

func Notify() error {
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
		case NotificationTypeEnum.String(E_simple_notification):
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
		case NotificationTypeEnum.String(e_recurring_notification):
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

func ShowNotifications(order string, sort string) error {
	ctx := context.Background()
	db, err := utils.DbConnect(ctx)
	if err != nil {
		return err
	}

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
		case NotificationTypeEnum.String(E_simple_notification):
			notification_details, err := queries.GetSimpleNotificationByNotificationId(ctx, notification.ID)
			if err != nil {
				return err
			}
			sb.WriteString(" at ")
			sb.WriteString(utils.LocalizeDateTime(notification_details.TriggerAt))
		case NotificationTypeEnum.String(e_recurring_notification):
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

func ShowNotificationDetails(notId int64) error {
	ctx := context.Background()
	db, err := utils.DbConnect(ctx)
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
	case E_simple_notification.String():
		notification_details, err := queries.GetSimpleNotificationByNotificationId(ctx, notification.ID)
		if err != nil {
			return err
		}
		sb.WriteString("\ttrigger_at: ")
		sb.WriteString(utils.LocalizeDateTime(notification_details.TriggerAt))
	case e_recurring_notification.String():
	}

	fmt.Println("Notification details:")
	fmt.Printf("\tid: %d\n", notification.ID)
	fmt.Printf("\tmessage_id: %d\n", notification.MessageID)
	fmt.Printf("\ttype: %s\n", notification.Type)
	fmt.Println(sb.String())
	fmt.Printf("\tcreated_at: %s\n", utils.LocalizeDateTime(notification.CreatedAt))
	fmt.Printf("\tupdated_at: %s\n", utils.LocalizeDateTime(notification.UpdatedAt))

	return nil
}

func CreateSimpleNotification (msgId int64, triggerAt time.Time) error {
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
		Type: NotificationTypeEnum.String(E_simple_notification),
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
		FeatureName: feat.E_notifications_feature.String(),
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	if exists == 1 {
		if err := qtx.IncrementMessageFeatureCount(ctx, sqlc.IncrementMessageFeatureCountParams{
			MessageID: msgId,
			FeatureName: feat.E_notifications_feature.String(),
		}); err != nil { return err }

		if err := tx.Commit(); err != nil { return err }

		return nil
	}

	if err = qtx.CreateMessageFeature(ctx, sqlc.CreateMessageFeatureParams{
		MessageID: msgId,
		FeatureName: feat.E_notifications_feature.String(),
	}); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func CreateRecurringNotification(msgId int64, weekDays []time.Weekday, triggerAt time.Time) error {
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
		Type: NotificationTypeEnum.String(e_recurring_notification),
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
		FeatureName: feat.E_notifications_feature.String(),
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	if exists == 1 {
		if err := qtx.IncrementMessageFeatureCount(ctx, sqlc.IncrementMessageFeatureCountParams{
			MessageID: msgId,
			FeatureName: feat.E_notifications_feature.String(),
		}); err != nil { return err }

		if err := tx.Commit(); err != nil { return err }

		return nil
	}

	if err = qtx.CreateMessageFeature(ctx, sqlc.CreateMessageFeatureParams{
		MessageID: msgId,
		FeatureName: feat.E_notifications_feature.String(),
	}); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func UpdateNotification(notId int64, triggerAt string, weekDays string) error {
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
	case E_simple_notification.String():
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

func DeleteNotification(notId int64) error {
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
		FeatureName: feat.E_notifications_feature.String(),
	}); err != nil {
		return err
	}

	return nil
}
