package tasks_manager

import (
	"context"
	"testing"

	"github.com/fatih/structs"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
)

func TestManager_pathConfigureCreateOrUpdate(t *testing.T) {
	ctx, b, _, storage := pathTestSetup(t)
	for _, op := range []logical.Operation{logical.CreateOperation, logical.UpdateOperation} {
		t.Run(string(op), func(t *testing.T) {
			t.Run("default", func(t *testing.T) {
				req := &logical.Request{
					Operation: op,
					Path:      "task/configure",
					Data:      make(map[string]interface{}),
					Storage:   storage,
				}

				resp, err := b.HandleRequest(ctx, req)
				assert.Nil(t, err)
				assert.Nil(t, resp)

				c, err := getConfiguration(ctx, storage)
				assert.Nil(t, err)
				assert.Equal(t, &configuration{
					TaskTimeout:      defaultTaskTimeoutValue,
					TaskHistoryLimit: defaultTaskHistoryLimit,
				}, c)
			})

			t.Run("custom", func(t *testing.T) {
				expectedTaskTimeout := "5m"
				expectedTaskHistoryLimit := 25

				req := &logical.Request{
					Operation: op,
					Path:      "task/configure",
					Data: map[string]interface{}{
						fieldNameTaskTimeout:      expectedTaskTimeout,
						fieldNameTaskHistoryLimit: expectedTaskHistoryLimit,
					},
					Storage: storage,
				}

				resp, err := b.HandleRequest(ctx, req)
				assert.Nil(t, err)
				assert.Nil(t, resp)

				c, err := getConfiguration(ctx, storage)
				assert.Nil(t, err)
				assert.Equal(t, &configuration{
					TaskTimeout:      expectedTaskTimeout,
					TaskHistoryLimit: expectedTaskHistoryLimit,
				}, c)
			})
		})
	}
}

func TestManager_pathConfigureRead(t *testing.T) {
	ctx, b, _, storage := pathTestSetup(t)

	t.Run("nonexistent", func(t *testing.T) {
		req := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "task/configure",
			Data:      make(map[string]interface{}),
			Storage:   storage,
		}

		resp, err := b.HandleRequest(ctx, req)
		assert.Nil(t, err)
		assert.Equal(t, errorResponseConfigurationNotFound, resp)
	})

	t.Run("normal", func(t *testing.T) {
		expectedConfig := &configuration{
			TaskTimeout:      "50h",
			TaskHistoryLimit: 1000,
		}
		err := putConfiguration(ctx, storage, expectedConfig)
		assert.Nil(t, err)

		req := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "task/configure",
			Data:      make(map[string]interface{}),
			Storage:   storage,
		}

		resp, err := b.HandleRequest(ctx, req)
		assert.Nil(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, structs.Map(expectedConfig), resp.Data)
	})
}

func pathTestSetup(t *testing.T) (context.Context, logical.Backend, Interface, logical.Storage) {
	ctx := context.Background()
	m := NewManager()
	storage := &logical.InmemStorage{}

	config := logical.TestBackendConfig()
	config.StorageView = storage

	b := &framework.Backend{Paths: m.Paths()}
	err := b.Setup(ctx, config)
	assert.Nil(t, err)

	return ctx, b, m, storage
}
