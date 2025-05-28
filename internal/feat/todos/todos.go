package todos

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/matheusbucater/gmess/internal/db/sqlc"
	// "github.com/matheusbucater/gmess/internal/feat"
	"github.com/matheusbucater/gmess/internal/utils"
)

type todoStatusEnum int
const (
	e_pending_status todoStatusEnum = iota
	e_done_status
)
var todoStatusName = map[todoStatusEnum]string{
	e_pending_status: "pending",
	e_done_status: 	  "done",
}
func (nte todoStatusEnum) string() string {
	return todoStatusName[nte]
}

func showTodos(order string, sort string) error {
	ctx := context.Background()
	db, err := utils.DbConnect(ctx)
	if err != nil { return err }

	queries := sqlc.New(db)
	
	todos := []sqlc.Todo{}
	switch order {
	case "created_at":
		if sort == "ASC" {
			todos, err = queries.GetTodosOrderByCreatedAtASC(ctx)
		} else {
			todos, err = queries.GetTodosOrderByCreatedAtDESC(ctx)
		}
	case "updated_at":
		if sort == "ASC" {
			todos, err = queries.GetTodosOrderByUpdatedAtASC(ctx)
		} else {
			todos, err = queries.GetTodosOrderByUpdatedAtDESC(ctx)
		}
	case "status":
		if sort == "ASC" {
			todos, err = queries.GetTodosOrderByStatusASC(ctx)
		} else {
			todos, err = queries.GetTodosOrderByStatusDESC(ctx)
		}
	}
	if err != nil { return err }
	
	todosCount := len(todos)

	if todosCount > 0 {
		fmt.Printf("(order by: '%s' %s)\n\n", order, sort)
	}

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
	db, err := utils.DbConnect(ctx)
	if err != nil {
		return err
	}

	queries := sqlc.New(db)

	exists, err := queries.TodoExists(ctx, todId)
	if err != nil { return err }
	if exists != 1 { return errors.New("todo does not exist") }

	todo, err := queries.GetTodoAndMessageByTodoId(ctx, todId)
	if err != nil { return err }

	fmt.Println("Todo details:")
	fmt.Printf("\tid: %d\n", todo.Todo.ID)
	fmt.Printf("\tmessage:\n")
	fmt.Printf("\t  id: %d\n", todo.Message.ID)
	fmt.Printf("\t  text: %s\n", todo.Message.Text)
	fmt.Printf("\t  created_at: %s\n", utils.LocalizeDateTime(todo.Message.CreatedAt))
	fmt.Printf("\t  updated_at: %s\n", utils.LocalizeDateTime(todo.Message.UpdatedAt))
	fmt.Printf("\tstatus: %s\n", todo.Todo.Status)
	fmt.Printf("\tcreated_at: %s\n", utils.LocalizeDateTime(todo.Todo.CreatedAt))
	fmt.Printf("\tupdated_at: %s\n", utils.LocalizeDateTime(todo.Todo.UpdatedAt))

	return nil
}

func createTodo(msgId int64) error {
	ctx := context.Background()
	db, err := utils.DbConnect(ctx)
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
		// FeatureName: feat.E_todos_feature.String(), 
		FeatureName: "todos", // TODO: fix this mess
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	if exists == 1 {
		if err := qtx.IncrementMessageFeatureCount(ctx, sqlc.IncrementMessageFeatureCountParams{
			MessageID: msgId,
			// FeatureName: feat.E_todos_feature.String(), 
			FeatureName: "todos", // TODO: fix this mess
		}); err != nil { return err }

		if err := tx.Commit(); err != nil { return err }

		return nil
	}

	if err = qtx.CreateMessageFeature(ctx, sqlc.CreateMessageFeatureParams{
		MessageID: msgId,
		// FeatureName: feat.E_todos_feature.String(), 
		FeatureName: "todos", // TODO: fix this mess
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
	db, err := utils.DbConnect(ctx)
	if err != nil { return err }

	queries := sqlc.New(db)

	exists, err := queries.TodoExists(ctx, todId)
	if err != nil { return err }
	if (exists == 0) { return errors.New("Invalid todo ID") }

	switch status {
	case e_pending_status.string():
		if _, err := queries.UpdateTodoById(ctx, sqlc.UpdateTodoByIdParams{
			ID: todId,
			Status: e_pending_status.string(),
		}); err != nil { return err }
	case e_done_status.string():
		if _, err := queries.UpdateTodoById(ctx, sqlc.UpdateTodoByIdParams{
			ID: todId,
			Status: e_done_status.string(),
		}); err != nil { return err }
	default:
		return fmt.Errorf("Invalid -status \"%s\"", status)
	}


	return nil
}

func deleteTodo(todId int64) error {
	ctx := context.Background()
	db, err := utils.DbConnect(ctx)
	if err != nil { return err }

	queries := sqlc.New(db)

	exists, err := queries.TodoExists(ctx, todId)
	if err != nil { return err }
	if (exists == 0) { return errors.New("Invalid todo ID") }

	msgId, err := queries.DeleteTodoByIdReturningMsgId(ctx, todId)
	if err = queries.DecrementMessageFeatureCount(ctx, sqlc.DecrementMessageFeatureCountParams{
		MessageID: msgId,
		// FeatureName: feat.E_todos_feature.String(), 
		FeatureName: "todos", // TODO: fix this mess
	}); err != nil { return err }

	if err = queries.DeleteTodoById(ctx, todId); err != nil { return err }

	return nil
}

func Cmd(args []string) {
	cmd := flag.NewFlagSet("todo", flag.ExitOnError)
	actionFlag := cmd.String("a", "r", "action:\n\t\"c\" create,\n\t\"r\" read,\n\t\"u\" update,\n\t\"d\" delete")
	msgIdFlag := cmd.Int64("msgId", -1, "message id")
	todIdFlag := cmd.Int64("todId", -1, "todo id")
	statusFlag := cmd.String("status", "", "todo status\n(pending,done)")
	orderFlag := cmd.String("order", "created_at", "order by: 'created_at', 'updated_at' or 'status'")
	descFlag := cmd.Bool("desc", false, "retrieve todos in descending order")

	if err := cmd.Parse(args); err != nil {
		fmt.Printf("error parsing cli args: %s\n", err)
		os.Exit(1)
	}

	switch *actionFlag {
	case "c":
		utils.EnforceRequiredFlags(cmd, []string{"msgId"})
		if err := createTodo(*msgIdFlag); err != nil {
			fmt.Printf("error creating todo: %s\n", err)
			os.Exit(1)
		}
	case "r":
		if *todIdFlag != -1 {
			if err := showTodoDetails(*todIdFlag); err != nil {
				fmt.Printf("error showing todos: %s\n", err)
				os.Exit(1)
			}
		} else {
			sort := "ASC"
			if *descFlag == true {
				sort = "DESC"
			}

			if !slices.Contains([]string{"created_at", "updated_at", "status"}, strings.ToLower(*orderFlag)) {
				fmt.Println("invalid value for '-order' flag")
				cmd.Usage()
				os.Exit(1)
			}
			if err := showTodos(strings.ToLower(*orderFlag), sort); err != nil {
				fmt.Printf("error showing todos: %s\n", err)
				os.Exit(1)
			}
		}
	case "u":
		utils.EnforceRequiredFlags(cmd, []string{"todId", "status"})
		if err := updateTodo(*todIdFlag, *statusFlag); err != nil {
			fmt.Printf("error updating todo: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("todo (%d) updated\n", *todIdFlag)
	case "d":
		utils.EnforceRequiredFlags(cmd, []string{"todId"})
		if err := deleteTodo(*todIdFlag); err != nil {
			fmt.Printf("error deleting todo: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("todo (%d) deleted\n", *todIdFlag)
	}
}
