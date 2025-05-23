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

func showMessageDetails(id int64) error {
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

	fmt.Printf("%d features available\n", featuresCount)
	for _, feat := range features {
		fmt.Printf("\n(%d) %s\n", feat.Seq, strings.ToLower(feat.Name))
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

func createSingleNotification (msgId int64) error {
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
		Type: "single",
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	err = qtx.CreateSingleNotification(ctx, sqlc.CreateSingleNotificationParams{
		NotificationID: notification.ID,
		TriggerAt: time.Now().Local().Add(5 * time.Minute),
	})
	if err != nil {
		tx.Rollback()
		return err
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

func main() {
	helloCmd := flag.NewFlagSet("hello", flag.ExitOnError)
	helloNameFlag := helloCmd.String("name", "", "name to be helloed")

	showCmd := flag.NewFlagSet("show", flag.ExitOnError)
	showFeaturesFlag := showCmd.Bool("feat", false, "show all features available")
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
	notifyMsgIdFlag := notifyCmd.Int64("msgId", -1, "id of the message to be notified")

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
		enforceRequiredFlags(notifyCmd, []string{"msgId"})
		err = createSingleNotification(*notifyMsgIdFlag)
		if err != nil {
			fmt.Printf("error creating notification: %s\n", err)
			os.Exit(1)
		}
	default:
		fmt.Println("expected 'hello', 'show', 'create', 'update', 'delete' or 'notify' subcommand.")
		os.Exit(1)
	}
}
