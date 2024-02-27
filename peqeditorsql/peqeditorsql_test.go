package peqeditorsql

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/xackery/talkeq/config"
)

func TestLogRotation(t *testing.T) {
	if os.Getenv("SINGLE_TEST") != "1" {
		t.Skip("skipping test; SINGLE_TEST not set")
	}
	client, err := New(context.Background(), config.PEQEditorSQL{
		IsEnabled:   true,
		Path:        ".",
		FilePattern: "sql_log_{{.Month}}-{{.Year}}.sql",
	})
	if err != nil {
		t.Fatalf("new client: %s", err)
	}

	path1 := fmt.Sprintf("sql_log_%s-%d.sql", time.Now().Format("01"), time.Now().Year())

	fmt.Println("test priming 'test'", path1)
	w1, err := os.Create(path1)
	if err != nil {
		t.Fatalf("create: %s", err)
	}
	defer func() {
		w1.Close()
		os.Remove(path1)
	}()
	w1.WriteString("test\n")
	err = client.Connect(context.Background())
	if err != nil {
		t.Fatalf("connect: %s", err)
	}

	time.Sleep(1 * time.Second)

	fmt.Println("test update 'test2'", path1)
	w1.WriteString("test2\n")

	time.Sleep(1 * time.Second)

	path2 := fmt.Sprintf("sql_log_%s-%d.sql", time.Now().AddDate(0, 1, 0).Format("01"), time.Now().Year())

	w2, err := os.Create(path2)
	if err != nil {
		t.Fatalf("create: %s", err)
	}
	defer func() {
		w2.Close()
		os.Remove(path2)
	}()

	fmt.Println("test writing 'test3'", path2)
	w2.WriteString("test3\n")

	time.Sleep(1 * time.Second)

	fmt.Println("test cleaning up")
	client.Disconnect(context.Background())
	//time.Sleep(1 * time.Second)
	fmt.Println("test finished")

}
