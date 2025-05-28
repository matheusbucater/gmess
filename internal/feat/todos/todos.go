package todos

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/matheusbucater/gmess/internal/db/sqlc"
	"github.com/matheusbucater/gmess/internal/feat"
	"github.com/matheusbucater/gmess/internal/utils"
)

type TodoStatusEnum int
const (
	E_pending_status TodoStatusEnum = iota
	E_done_status
)
var todoStatusName = map[TodoStatusEnum]string{
	E_pending_status: "pending",
	E_done_status: 	  "done",
}
func (nte TodoStatusEnum) String() string {
	return todoStatusName[nte]
}

func ShowTodos(order string, sort string) error {
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

func ShowTodoDetails(todId int64) error {
	ctx := context.Background()
	db, err := utils.DbConnect(ctx)
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
	fmt.Printf("\tcreated_at: %s\n", utils.LocalizeDateTime(todo.CreatedAt))
	fmt.Printf("\tupdated_at: %s\n", utils.LocalizeDateTime(todo.UpdatedAt))

	return nil
}

func CreateTodo(msgId int64) error {
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
		FeatureName: feat.E_todos_feature.String(),
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	if exists == 1 {
		if err := qtx.IncrementMessageFeatureCount(ctx, sqlc.IncrementMessageFeatureCountParams{
			MessageID: msgId,
			FeatureName: feat.E_todos_feature.String(),
		}); err != nil { return err }

		if err := tx.Commit(); err != nil { return err }

		return nil
	}

	if err = qtx.CreateMessageFeature(ctx, sqlc.CreateMessageFeatureParams{
		MessageID: msgId,
		FeatureName: feat.E_todos_feature.String(),
	}); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func UpdateTodo(todId int64, status string) error {
	ctx := context.Background()
	db, err := utils.DbConnect(ctx)
	if err != nil { return err }

	queries := sqlc.New(db)

	exists, err := queries.TodoExists(ctx, todId)
	if err != nil { return err }
	if (exists == 0) { return errors.New("Invalid todo ID") }

	switch status {
	case E_pending_status.String():
		if _, err := queries.UpdateTodoById(ctx, sqlc.UpdateTodoByIdParams{
			ID: todId,
			Status: E_pending_status.String(),
		}); err != nil { return err }
	case E_done_status.String():
		if _, err := queries.UpdateTodoById(ctx, sqlc.UpdateTodoByIdParams{
			ID: todId,
			Status: E_done_status.String(),
		}); err != nil { return err }
	default:
		return fmt.Errorf("Invalid -status \"%s\"", status)
	}


	return nil
}

func DeleteTodo(todId int64) error {
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
		FeatureName: feat.E_todos_feature.String(),
	}); err != nil { return err }

	if err = queries.DeleteTodoById(ctx, todId); err != nil { return err }

	return nil
}
