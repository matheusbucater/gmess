package feat

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/matheusbucater/gmess/internal/db/sqlc"
	"github.com/matheusbucater/gmess/internal/utils"
)

type FeatureEnum int
const (
	E_notifications_feature FeatureEnum = iota
	E_todos_feature 
	E_feature_not_available
)
var featureName = map[FeatureEnum]string{
	E_notifications_feature: "notifications",
	E_todos_feature: "todos",
	E_feature_not_available: "not_available",
}
func (fe FeatureEnum) String() string {
	return featureName[fe]
}

func FeatureExists(feat FeatureEnum) (bool, error) {
	ctx := context.Background()
	db, err := utils.DbConnect(ctx)
	if err != nil {
		return false, err
	}
	queries := sqlc.New(db)

	exists, err := queries.FeatureExists(ctx, feat.String())
	return exists == 1, nil
}

func ShowFeatures() error {
	ctx := context.Background()
	db, err := utils.DbConnect(ctx)
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
