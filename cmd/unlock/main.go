package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	// include dot env loader
	_ "github.com/joho/godotenv/autoload"
)

type config struct {
	tableName string
	endpoint  string
	region    string
	accessKey string
	secretKey string

	releaseLocks bool
}

// lockInfo is the information about a lock that is stored in the lock table.
// https://github.com/hashicorp/terraform/blob/3bea1171aff32504ea5e95ba7b129f35f8d92cd8/internal/states/statemgr/locker.go#L118C1-L141C2
type lockInfo struct {
	// Unique ID for the lock. NewLockInfo provides a random ID, but this may
	// be overridden by the lock implementation. The final value of ID will be
	// returned by the call to Lock.
	ID string

	LockID string

	// Terraform operation, provided by the caller.
	Operation string

	// Extra information to store with the lock, provided by the caller.
	Info string

	// user@hostname when available
	Who string

	// Terraform version
	Version string

	// Time that the lock was taken.
	Created time.Time

	// Path to the state file when applicable. Set by the Lock implementation.
	Path string
}

func getConfigs() config {
	return config{
		tableName:    os.Getenv("DYNAMODB_TABLE_NAME"),
		endpoint:     os.Getenv("DYNAMODB_ENDPOINT"),
		region:       os.Getenv("AWS_REGION"),
		accessKey:    os.Getenv("AWS_ACCESS_KEY_ID"),
		secretKey:    os.Getenv("AWS_SECRET_ACCESS_KEY"),
		releaseLocks: strings.ToLower(os.Getenv("RELEASE_LOCKS")) == "true",
	}
}

func main() {
	conf := getConfigs()

	fmt.Printf("release locks: %t\n", conf.releaseLocks)

	db := dynamodb.New(dynamodb.Options{
		BaseEndpoint: aws.String(conf.endpoint),
		Region:       conf.region,
		Credentials: credentials.NewStaticCredentialsProvider(
			conf.accessKey,
			conf.secretKey,
			"",
		),
	})

	locks, err := getLocks(db, conf.tableName)
	if err != nil {
		panic(err)
	}

	if len(locks) == 0 {
		fmt.Println("✅ no locks found")
		return
	}

	fmt.Printf("found %d locks\n", len(locks))

	for _, lock := range locks {
		fmt.Printf("LockID: %s; ID: %s; %s; Holder: %s; Time: %s\n", lock.LockID, lock.ID, lock.Operation, lock.Who, lock.Created)
		if conf.releaseLocks {
			err := unlock(db, conf.tableName, lock.ID, lock.LockID)
			if err != nil {
				fmt.Printf("❌ failed to unlock %s: %s\n", lock.LockID, err)
				continue
			}
			fmt.Printf("✅ unlocked %s\n", lock.LockID)
		}
	}

}

// getLocks returns a list of locks from the dynamodb table
func getLocks(db *dynamodb.Client, tableName string) ([]lockInfo, error) {
	// implement getting terraform locks from dynamodb table

	outputs, err := db.Scan(context.Background(), &dynamodb.ScanInput{
		TableName:            aws.String(tableName),
		ProjectionExpression: aws.String("LockID, Info"),
	})

	if err != nil {
		return nil, fmt.Errorf("dynamodb scan: %w", err)
	}

	locks := []lockInfo{}

	for _, item := range outputs.Items {
		var lock lockInfo
		if v, ok := item["Info"].(*types.AttributeValueMemberS); ok && v.Value != "" {
			err := json.Unmarshal([]byte(v.Value), &lock)
			if err != nil {
				return nil, fmt.Errorf("json unmarshal: %w", err)
			}
			lock.LockID = item["LockID"].(*types.AttributeValueMemberS).Value
			locks = append(locks, lock)
		}

	}

	return locks, nil
}

func getLockInfo(db *dynamodb.Client, tableName, id string) (*lockInfo, error) {
	getParams := &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"LockID": &types.AttributeValueMemberS{Value: id},
		},
		ProjectionExpression: aws.String("LockID, Info"),
		TableName:            aws.String(tableName),
		ConsistentRead:       aws.Bool(true),
	}

	resp, err := db.GetItem(context.Background(), getParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	var infoData string
	if v, ok := resp.Item["Info"].(*types.AttributeValueMemberS); ok && v.Value != "" {
		infoData = v.Value
	}

	lockInfo := &lockInfo{}
	err = json.Unmarshal([]byte(infoData), lockInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %w", infoData, err)
	}

	return lockInfo, nil
}

func unlock(db *dynamodb.Client, tableName, id, path string) error {

	lockInfo, err := getLockInfo(db, tableName, path)
	if err != nil {
		return fmt.Errorf("failed to retrieve lock info: %s", err)
	}

	if lockInfo.ID != id {
		return fmt.Errorf("lock id %q does not match existing lock", id)
	}

	params := &dynamodb.DeleteItemInput{
		Key: map[string]types.AttributeValue{
			"LockID": &types.AttributeValueMemberS{Value: path},
		},
		TableName: aws.String(tableName),
	}
	_, err = db.DeleteItem(context.Background(), params)

	if err != nil {
		return fmt.Errorf("failed to delete item from dynamo db: %w", err)
	}

	return nil
}
