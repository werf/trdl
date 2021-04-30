package tasks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/tasks/queue"
)

const (
	storageKeyCurrentRunningTask = "current_running_task"
	storageKeyPrefixTask         = "task-"
	storageKeyPrefixTaskLog      = "task_log-"

	taskStatusRunning   = "RUNNING"
	taskStatusSucceeded = "SUCCEEDED"
	taskStatusFailed    = "FAILED"
	taskStatusCanceled  = "CANCELED"
)

type Manager struct {
	Storage logical.Storage
	Queue   *queue.Queue

	mu sync.Mutex
}

func newManager() *Manager {
	return &Manager{}
}

func (m *Manager) initManager(storage logical.Storage) {
	if m.Storage != nil {
		return
	}

	m.Storage = storage
	m.initQueue()
}

func (m *Manager) restartQueue() {
	if m.Queue != nil {
		m.Queue.Stop()
	}

	m.initQueue()
}

func (m *Manager) initQueue() {
	m.Queue = queue.NewQueue(queue.Callbacks{
		TaskStartedCallback:   m.taskStartedCallback,
		TaskFailedCallback:    m.taskFailedCallback,
		TaskCompletedCallback: m.taskCompletedCallback,
	})
	m.Queue.Start()
}

func (m *Manager) HasRunningTask(uuid string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Queue == nil {
		return false
	}

	return m.Queue.HasRunningTask(uuid)
}

func (m *Manager) GetTaskLog(uuid string) []byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Queue == nil {
		return nil
	}

	return m.Queue.GetTaskLog(uuid)
}

func (m *Manager) RunOptionalTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(ctx context.Context, storage logical.Storage) error) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.initManager(reqStorage) // initialize on first call

	queueTaskFunc := func(ctx context.Context) error {
		return taskFunc(ctx, m.Storage)
	}

	return m.Queue.AddOptionalTask(ctx, queueTaskFunc)
}

func (m *Manager) RunTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(ctx context.Context, storage logical.Storage) error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.initManager(reqStorage) // initialize on first call

	queueTaskFunc := func(ctx context.Context) error {
		return taskFunc(ctx, m.Storage)
	}

	m.Queue.AddTask(ctx, queueTaskFunc)
}

func (m *Manager) taskStartedCallback(ctx context.Context, uuid string) error {
	if err := markStaleTaskAsFailed(ctx, m.Storage); err != nil {
		return err
	}

	if err := m.Storage.Put(ctx, &logical.StorageEntry{
		Key:   storageKeyCurrentRunningTask,
		Value: []byte(uuid),
	}); err != nil {
		return err
	}

	task := &Task{}
	task.UUID = uuid
	task.Status = taskStatusRunning

	tNow := time.Now()
	task.Created = tNow
	task.Modified = tNow

	if err := putTaskIntoStorage(ctx, m.Storage, task); err != nil {
		return err
	}

	return nil
}

func (m *Manager) taskCompletedCallback(ctx context.Context, uuid string, log []byte) error {
	task, err := getTaskFromStorage(ctx, m.Storage, uuid)
	if err != nil {
		return err
	}

	if task == nil {
		panic(fmt.Sprintf("unexpected error: task %q not found in storage", uuid))
	}

	task.Status = taskStatusSucceeded
	task.Modified = time.Now()
	if err := putTaskIntoStorage(ctx, m.Storage, task); err != nil {
		return err
	}

	if err := m.Storage.Put(ctx, &logical.StorageEntry{
		Key:   taskLogStorageKey(uuid),
		Value: log,
	}); err != nil {
		return err
	}

	if err := m.Storage.Delete(ctx, storageKeyCurrentRunningTask); err != nil {
		return err
	}

	return nil
}

func (m *Manager) taskFailedCallback(ctx context.Context, uuid string, log []byte, taskErr error) error {
	task, err := getTaskFromStorage(ctx, m.Storage, uuid)
	if err != nil {
		return err
	}

	if task == nil {
		panic(fmt.Sprintf("unexpected error: task %q not found in storage", uuid))
	}

	task.Status = taskStatusFailed
	task.Modified = time.Now()
	task.Reason = taskErr.Error()
	if err := putTaskIntoStorage(ctx, m.Storage, task); err != nil {
		return err
	}

	if err := m.Storage.Put(ctx, &logical.StorageEntry{
		Key:   taskLogStorageKey(uuid),
		Value: log,
	}); err != nil {
		return err
	}

	if err := m.Storage.Delete(ctx, storageKeyCurrentRunningTask); err != nil {
		return err
	}

	return nil
}

func (m *Manager) CancelRunningTask(uuid string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.HasRunningTask(uuid) {
		return
	}

	m.restartQueue()
}

func markStaleTaskAsFailed(ctx context.Context, storage logical.Storage) error {
	uuid, err := getCurrentTaskUUIDFromStorage(ctx, storage)
	if err != nil {
		return err
	}

	if uuid == "" {
		return nil
	}

	task, err := getTaskFromStorage(ctx, storage, uuid)
	if err != nil {
		return err
	}

	if task == nil {
		return nil
	}

	task.Status = taskStatusFailed
	task.Modified = time.Now()
	task.Reason = "the unfinished task from the previous run"
	if err := putTaskIntoStorage(ctx, storage, task); err != nil {
		return err
	}

	return nil
}

func markTaskAsCanceled(ctx context.Context, storage logical.Storage, uuid string) error {
	task, err := getTaskFromStorage(ctx, storage, uuid)
	if err != nil {
		return err
	}

	if task == nil {
		panic(fmt.Sprintf("unexpected error: task %q not found in storage", uuid))
	}

	task.Status = taskStatusCanceled
	task.Modified = time.Now()
	if err := putTaskIntoStorage(ctx, storage, task); err != nil {
		return err
	}

	return nil
}

func getCurrentTaskUUIDFromStorage(ctx context.Context, storage logical.Storage) (string, error) {
	currentRunningTaskValue, err := storage.Get(ctx, storageKeyCurrentRunningTask)
	if err != nil {
		return "", err
	}

	if currentRunningTaskValue == nil {
		return "", err
	}

	return string(currentRunningTaskValue.Value), nil
}

func getTaskFromStorage(ctx context.Context, storage logical.Storage, uuid string) (*Task, error) {
	e, err := storage.Get(ctx, taskStorageKey(uuid))
	if err != nil {
		return nil, err
	}

	if e == nil {
		return nil, nil
	}

	return storageEntryToTask(e)
}

func getTaskLogFromStorage(ctx context.Context, storage logical.Storage, uuid string) ([]byte, error) {
	e, err := storage.Get(ctx, taskLogStorageKey(uuid))
	if err != nil {
		return nil, err
	}

	if e == nil {
		return nil, nil
	}

	return e.Value, nil
}

func putTaskIntoStorage(ctx context.Context, storage logical.Storage, task *Task) error {
	e, err := taskToStorageEntry(task)
	if err != nil {
		return err
	}

	return storage.Put(ctx, e)
}
