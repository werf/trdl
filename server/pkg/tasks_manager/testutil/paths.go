package testutil

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
)

func ListTasks(t TInterface, ctx context.Context, b logical.Backend, storage logical.Storage) []string {
	req := &logical.Request{
		Operation:  logical.ReadOperation,
		Path:       "task",
		Storage:    storage,
		Connection: &logical.Connection{},
	}

	resp, err := b.HandleRequest(ctx, req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%v resp:%#v\n", err, resp)
	}

	return resp.Data["keys"].([]string)
}

func GetTaskStatus(t TInterface, ctx context.Context, b logical.Backend, storage logical.Storage, uuid string) (string, string) {
	req := &logical.Request{
		Operation:  logical.ReadOperation,
		Path:       fmt.Sprintf("task/%s", uuid),
		Storage:    storage,
		Connection: &logical.Connection{},
	}

	resp, err := b.HandleRequest(ctx, req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%v resp:%#v\n", err, resp)
	}

	return resp.Data["status"].(string), resp.Data["reason"].(string)
}

func GetTaskLog(t TInterface, ctx context.Context, b logical.Backend, storage logical.Storage, uuid string) string {
	req := &logical.Request{
		Operation:  logical.ReadOperation,
		Path:       fmt.Sprintf("task/%s/log", uuid),
		Storage:    storage,
		Connection: &logical.Connection{},
		Data: map[string]interface{}{
			"limit": 1000_000_000,
		},
	}

	resp, err := b.HandleRequest(ctx, req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%v resp:%#v\n", err, resp)
	}

	return resp.Data["result"].(string)
}

func WaitForTaskCompletion(w io.Writer, t TInterface, ctx context.Context, b logical.Backend, storage logical.Storage, uuid string) {
	for {
		status, reason := GetTaskStatus(t, ctx, b, storage, uuid)

		_, _ = fmt.Fprintf(w, "Poll task %s: status=%s reason=%q\n", uuid, status, reason)

		switch status {
		case "QUEUED", "RUNNING":
		case "COMPLETED", "FAILED":
			return
		default:
			taskLog := GetTaskLog(t, ctx, b, storage, uuid)
			t.Fatalf("got unexpected task %s status %s reason %s:\n%s\n", uuid, status, reason, taskLog)
		}

		time.Sleep(1 * time.Second)
	}
}

func WaitForTaskSuccess(w io.Writer, t TInterface, ctx context.Context, b logical.Backend, storage logical.Storage, uuid string) {
	for {
		status, reason := GetTaskStatus(t, ctx, b, storage, uuid)

		_, _ = fmt.Fprintf(w, "Poll task %s: status=%s reason=%q\n", uuid, status, reason)

		switch status {
		case "QUEUED", "RUNNING":
		case "SUCCEEDED":
			return
		default:
			taskLog := GetTaskLog(t, ctx, b, storage, uuid)
			t.Fatalf("got unexpected task %s status %s reason %s:\n%s\n", uuid, status, reason, taskLog)
		}

		time.Sleep(1 * time.Second)
	}
}

func WaitForTaskFailure(w io.Writer, t TInterface, ctx context.Context, b logical.Backend, storage logical.Storage, uuid string) string {
	for {
		status, reason := GetTaskStatus(t, ctx, b, storage, uuid)

		_, _ = fmt.Fprintf(w, "Poll task %s: status=%s reason=%q\n", uuid, status, reason)

		switch status {
		case "QUEUED", "RUNNING":
		case "FAILED":
			return reason
		default:
			taskLog := GetTaskLog(t, ctx, b, storage, uuid)
			t.Fatalf("got unexpected task %s status %s reason %s:\n%s\n", uuid, status, reason, taskLog)
		}

		time.Sleep(1 * time.Second)
	}
}

// TInterface covers most of the methods in the testing package's T.
type TInterface interface {
	Cleanup(func())
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fail()
	FailNow()
	Failed() bool
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Helper()
	Log(args ...interface{})
	Logf(format string, args ...interface{})
	Name() string
	Parallel()
	Skip(args ...interface{})
	SkipNow()
	Skipf(format string, args ...interface{})
	Skipped() bool
	TempDir() string
}
