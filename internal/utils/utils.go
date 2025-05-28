package utils

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
)

var ddl string

func LocalizeDateTime(datetime time.Time) string {
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

func EnforceRequiredFlags(cmd *flag.FlagSet, required []string) {
	seen := make(map[string]bool)
	cmd.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	for _, req := range required {
		if !seen[req] {
			fmt.Printf("missing required '-%s' flag.\n", req)
			os.Exit(1)
		}
	}
}

func DbConnect(ctx context.Context) (*sql.DB, error) {
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

func ParseWeekDays(wdString string) ([]time.Weekday, error) {
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
