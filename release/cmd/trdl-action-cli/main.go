package main

import (
	"log"
	"os"

	"github.com/werf/trdl/release/pkg/vault"
)

func main() {
	vaultToken := os.Getenv("VAULT_TOKEN")
	projectName := os.Getenv("TRDL_RELEASE_PROJECT_NAME")
	gitTag := os.Getenv("TRDL_GIT_TAG")

	client, err := vault.NewTrdlClient(vaultToken)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	taskLogger := func(taskID, msg string) {
		log.Printf("[%s] %s", taskID, msg)
	}

	switch os.Getenv("TRDL_OPERATION") {
	case "publish":
		log.Println("Starting publish...")
		err = client.Publish(projectName, taskLogger)
	case "release":
		log.Println("Starting release...")
		err = client.Release(projectName, gitTag, taskLogger)
	default:
		log.Fatalf("Unknown operation. Set TRDL_OPERATION to 'publish' or 'release'.")
	}

	if err != nil {
		log.Fatalf("Operation failed: %v", err)
	}

	log.Println("Operation completed successfully!")
}
