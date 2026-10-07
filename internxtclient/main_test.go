package internxtclient_test

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	client "github.com/StarHack/go-internxt-drive/internxtclient"
)

var (
	testEmail      string
	testPassword   string
	testCtx               = context.Background()
	testBytes      []byte = []byte{0x13, 0x09, 0x20, 0x23}
	c              *client.Client
	testFolderUUID string
	TESTFOLDER     = "go-internxt-drive-tests"
)

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		// Unit tests run offline; integration tests skip themselves in short mode.
		os.Exit(m.Run())
	}

	testEmail = os.Getenv("INTERNXT_TEST_EMAIL")
	testPassword = os.Getenv("INTERNXT_TEST_PASSWORD")

	if testEmail == "" || testPassword == "" {
		log.Fatal("Skipping tests: required environment variables not set.\nPlease set INTERNXT_TEST_EMAIL and INTERNXT_TEST_PASSWORD environment variables.")
	}

	//Setting a random name for each run
	b := make([]byte, 2)
	_, _ = rand.Read(b)

	TESTFOLDER = TESTFOLDER + "-" + fmt.Sprintf("%x", b)

	println("Setting up environment in " + TESTFOLDER)
	err := setupCreateTestEnvironment()
	time.Sleep(1 * time.Second)
	if err != nil {
		fmt.Printf("Couldn't setup testing environment")
		println("Tearing down environment")
		err = setupTeardownTestEnvironment()
		if err != nil {
			log.Fatalf("Couldn't tear down testing environment: %v", err)
		}
		log.Fatalf("Skipping tests: couldn't set up testing environment: %v", err)
	}

	println("Test environment set up in folder with uuid: " + testFolderUUID)

	code := m.Run()

	println("Tearing down environment")
	err = setupTeardownTestEnvironment()
	if err != nil {
		log.Fatalf("Couldn't tear down testing environment: %v", err)
	}

	os.Exit(code)
}

func setupCreateTestEnvironment() error {
	err := setupClient()
	if err != nil {
		return err
	}

	err = setupCreateFolderStructure()
	if err != nil {
		return err
	}

	return nil
}

func setupTeardownTestEnvironment() error {
	err := setupPurgeFolderStructure()
	if err != nil {
		return err
	}

	return nil
}

func setupClient() error {
	cl, err := client.NewWithCredentials(testCtx, testEmail, testPassword)
	if err != nil {
		return fmt.Errorf("Couldn't set up testing client: %v", err)
	}

	c = cl

	return nil
}

func setupCreateFolderStructure() error {
	createFolderRequest := client.CreateFolderRequest{
		PlainName:        TESTFOLDER,
		ParentFolderUUID: c.UserData.AccessData.User.RootFolderUUID,
	}
	folder, err := c.Folders.CreateFolder(testCtx, createFolderRequest)
	if err != nil {
		return fmt.Errorf("Couldn't create test folder: %v", err)
	}
	testFolderUUID = folder.UUID
	return nil
}

func setupPurgeFolderStructure() error {
	if testFolderUUID != "" {
		err := c.Folders.DeleteFolder(testCtx, testFolderUUID)
		if err != nil {
			return fmt.Errorf("Couldn't purge test folder: %v", err)
		}
		return nil
	}
	return fmt.Errorf("no testFolder to delete")
}
