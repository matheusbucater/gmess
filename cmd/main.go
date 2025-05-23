package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
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
		"\tid: %d\n\ttext: %s\n\tcreated_at: %s\n\tupdated_at: %s\n", 
		message.ID, message.Text,
		localizeDateTime(message.CreatedAt),
		localizeDateTime(message.UpdatedAt),
	)

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

	messagesCount, err := queries.CountMessages(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("(order by: '%s' %s)\n\n", order, sort)
	fmt.Printf("You have %d messages:\n\n", messagesCount)
	for _, message := range messages {
		fmt.Printf(
			"(%d) %s\n", 
			message.ID,
			message.Text,
		)
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

func main() {
	helloCmd := flag.NewFlagSet("hello", flag.ExitOnError)
	helloNameFlag := helloCmd.String("name", "", "name to be helloed")

	showCmd := flag.NewFlagSet("show", flag.ExitOnError)
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

	if len(os.Args) < 2 {
		fmt.Println("expected 'hello', 'show', 'create', 'update' or 'delete' subcommand.")
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
	default:
		fmt.Println("expected 'hello', 'show' or 'create' subcommand.")
		os.Exit(1)
	}
}
