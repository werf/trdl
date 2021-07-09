package tasks_manager

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fatih/structs"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
)

func TestManager_pathConfigureCreateOrUpdate(t *testing.T) {
	for _, op := range []logical.Operation{logical.CreateOperation, logical.UpdateOperation} {
		t.Run(string(op), func(t *testing.T) {
			t.Run("default", func(t *testing.T) {
				ctx, b, _, storage := pathTestSetup(t)

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
					TaskTimeout:      defaultTaskTimeoutDuration,
					TaskHistoryLimit: fieldDefaultTaskHistoryLimit,
				}, c)
			})

			t.Run("custom", func(t *testing.T) {
				ctx, b, _, storage := pathTestSetup(t)

				fieldValueTaskTimeout := "5m"
				fieldValueTaskHistoryLimit := 25
				expectedTaskTimeout := 5 * time.Minute
				expectedTaskHistoryLimit := fieldValueTaskHistoryLimit

				req := &logical.Request{
					Operation: op,
					Path:      "task/configure",
					Data: map[string]interface{}{
						fieldNameTaskTimeout:      fieldValueTaskTimeout,
						fieldNameTaskHistoryLimit: fieldValueTaskHistoryLimit,
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

			t.Run("invalid fields", func(t *testing.T) {
				for _, test := range []struct {
					name             string
					taskTimeout      interface{}
					taskHistoryLimit interface{}
					expectedErrMsg   string
				}{
					{
						name:           fmt.Sprintf("invalid %s", fieldNameTaskTimeout),
						taskTimeout:    "no valid",
						expectedErrMsg: fmt.Sprintf("error converting input no valid for field %q: strconv.ParseInt: parsing \"no valid\": invalid syntax", fieldNameTaskTimeout),
					},
					{
						name:             fmt.Sprintf("invalid %s", fieldNameTaskHistoryLimit),
						taskHistoryLimit: "no valid",
						expectedErrMsg:   fmt.Sprintf("error converting input no valid for field %q: cannot parse '' as int: strconv.ParseInt: parsing \"no valid\": invalid syntax", fieldNameTaskHistoryLimit),
					},
				} {
					t.Run(test.name, func(t *testing.T) {
						ctx, b, _, storage := pathTestSetup(t)

						data := make(map[string]interface{})
						if test.taskTimeout != nil {
							data[fieldNameTaskTimeout] = test.taskTimeout
						}

						if test.taskHistoryLimit != nil {
							data[fieldNameTaskHistoryLimit] = test.taskHistoryLimit
						}

						req := &logical.Request{
							Operation: op,
							Path:      "task/configure",
							Data:      data,
							Storage:   storage,
						}

						// check field validation failed
						resp, err := b.HandleRequest(ctx, req)
						assert.Nil(t, resp)
						assert.NotNil(t, err)
						if assert.Error(t, err) {
							assert.Equal(t, test.expectedErrMsg, err.Error())
						}

						// check no configuration in storage
						c, err := getConfiguration(ctx, storage)
						assert.Nil(t, err)
						assert.Nil(t, c)
					})
				}
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
			TaskTimeout:      50 * time.Hour,
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
