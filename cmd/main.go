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
	"github.com/matheusbucater/gmess/internal/utils"

	_ "modernc.org/sqlite"
)

func showMessages(order string, sort string) error {
	ctx := context.Background()
	db, err := utils.DbConnect(ctx)
	if err != nil { return err }

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
	if err != nil { return err }

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
		features, err := queries.GetPrettyFeaturesByMessageId(ctx, message.ID)
		if err != nil { return err }

		fmt.Printf("(%d) %s", message.ID, message.Text)
		if len(features) > 0 { fmt.Printf(" [%s]", features) }
		fmt.Println()
	}
	return nil
}

func showMessageDetails(id int64) error {
	ctx := context.Background()
	db, err := utils.DbConnect(ctx)
	if err != nil { return err }

	queries := sqlc.New(db)

	exists, err := queries.MessageExists(ctx, id)
	if err != nil { return err }
	if exists == 0 { return errors.New("Invalid message ID") }

	message_details, err := queries.GetMessageAndFeatures(ctx, id)
	if err != nil { return err }

	fmt.Println("Message details:")
	fmt.Printf(
		"\tid: %d\n\ttext: %s\n\tcreated_at: %s\n\tupdated_at: %s\n\tfeatures: %s\n", 
		message_details.ID, message_details.Text,
		utils.LocalizeDateTime(message_details.CreatedAt),
		utils.LocalizeDateTime(message_details.UpdatedAt),
		message_details.Features,
	)

	return nil
}

func createMessage(message string) error {
	ctx := context.Background()
	db, err := utils.DbConnect(ctx)
	if err != nil { return err }

	queries := sqlc.New(db)

	if _, err := queries.CreateMessage(ctx, message); err != nil {
		return err
	}
	return nil
}

func updateMessage(id int64, message string) error {
	ctx := context.Background()
	db, err := utils.DbConnect(ctx)
	if err != nil { return err }

	queries := sqlc.New(db)

	exists, err := queries.MessageExists(ctx, id)
	if (exists == 0) { return errors.New("Invalid message ID") }
	
	if _, err = queries.UpdateMessage(ctx, sqlc.UpdateMessageParams{ ID: id, Text: message }); err != nil { return err }
	return nil
}

func deleteMessage(id int64) error {
	ctx := context.Background()
	db, err := utils.DbConnect(ctx)
	if err != nil { return err }

	queries := sqlc.New(db)

	exists, err := queries.MessageExists(ctx, id)
	if err != nil { return err }
	if (exists == 0) { return errors.New("Invalid message ID") }
	
	if err = queries.DeleteMessage(ctx, id); err != nil { return err }
	return nil
}

func main() {
	time.Local, _ = time.LoadLocation("America/Sao_Paulo")

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

	if len(os.Args) < 2 {
		fmt.Println("expected 'hello', 'show', 'create', 'update', 'delete' or [feature] subcommand.")
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
		if err := createCmd.Parse(os.Args[2:]); err != nil {
			fmt.Printf("error parsing cli args: %s\n", err)
		}
		utils.EnforceRequiredFlags(createCmd, []string{"message"})
		if err := createMessage(*createMessageFlag); err != nil {
			fmt.Printf("error creating new message: %s\n", err)
			os.Exit(1)
		}
		fmt.Println("message created.")
	case "update":
		err := updateCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Printf("error parsing cli args: %s\n", err)
			os.Exit(1)
		}
		utils.EnforceRequiredFlags(updateCmd, []string{"id", "message"})
		if err = updateMessage(*updateIdFlag, *updateMessageFlag); err != nil {
			fmt.Printf("error updating message: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("message (%d) udpated\n", *updateIdFlag)
	case "delete":
		if err := deleteCmd.Parse(os.Args[2:]); err != nil {
			fmt.Printf("error parsing cli args: %s\n", err)
			os.Exit(1)
		}
		utils.EnforceRequiredFlags(deleteCmd, []string{"id"})
		if err := deleteMessage(*deleteIdFlag); err != nil {
			fmt.Printf("error deleting message: %s\n", err)
			os.Exit(1)
		}
		fmt.Println("Message deleted.")
	default:
		exists, err := feat.FeatureExists(os.Args[1])
		if err != nil {
			fmt.Printf("error checking feature existence: %s\n", err)
			os.Exit(1)
		}
		if !exists {
			fmt.Printf("feature \"%s\" not available.\n", os.Args[1])
			flag.Usage()
			os.Exit(1)
		}

		if err := feat.HandleCmd(os.Args[1], os.Args[2:]); err != nil {
			fmt.Printf("error handling feature command: %s\n", err)
			os.Exit(1)
		}
	}
}
