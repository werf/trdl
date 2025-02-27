package server

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/logboek"
	"github.com/werf/trdl/server/pkg/publisher"
	"github.com/werf/trdl/server/pkg/tasks_manager"
	"github.com/werf/trdl/server/pkg/util"
)

var SystemClock util.Clock = util.NewSystemClock()

const (
	// TODO: Do not use global run timestamp to specify repository operations period.
	// TODO: Instead let publisher.Repository decide whether or not it is appropriate time
	// TODO: to update timestamp, or to rotate private keys.
	// TODO: Publisher.RotatePrivKeys may be called every minute, but it will actually rotate keys
	// TODO: when it is appropriate time to rotate (based on each private key expiration date — which is internal data of publisher package).
	// TODO: The same with publisher.UpdateTimestamps — let this procedure and internal Repository object decide when to update timestamps
	// TODO: based on timestamp.json data.
	// TODO:
	// TODO: For now timestamps being updated every hour forcefully and private keys are not rotated yet (will be implemented soon).
	lastPeriodicRunTimestampKey = "last_periodic_run_timestamp"
	periodicRunPeriod           = 1 * time.Hour
)

func (b *Backend) Periodic(ctx context.Context, req *logical.Request) error {
	entry, err := req.Storage.Get(ctx, lastPeriodicRunTimestampKey)
	if err != nil {
		return fmt.Errorf("unable to get key %q from storage: %w", lastPeriodicRunTimestampKey, err)
	}

	if entry != nil {
		lastRunTimestamp, err := strconv.ParseInt(string(entry.Value), 10, 64)
		if err == nil && SystemClock.Since(time.Unix(lastRunTimestamp, 0)) < periodicRunPeriod {
			b.Logger().Info("Waiting rotate repository keys period: skipping periodic task")
			return nil
		}
	}

	config, err := getConfiguration(ctx, req.Storage)
	if err != nil {
		return fmt.Errorf("unable to get configuration: %w", err)
	}
	if config == nil {
		b.Logger().Info("Configuration not set: skipping periodic task")
		return nil
	}

	{
		cfgData, err := config.maskConfigSensetiveDataForDebug()
		b.Logger().Debug(fmt.Sprintf("Got configuration (err=%v):\n%s", err, cfgData))
	}

	opts := config.RepositoryOptions()
	opts.InitializeTUFKeys = false
	opts.InitializePGPSigningKey = true
	publisherRepository, err := b.Publisher.GetRepository(ctx, req.Storage, opts)
	if err == publisher.ErrUninitializedRepositoryKeys {
		b.Logger().Info("Repository is not initialized: skipping periodic task")
		return nil
	}
	if err != nil {
		return fmt.Errorf("error getting publisher repository: %w", err)
	}

	now := SystemClock.Now()
	uuid, err := b.TasksManager.RunTask(ctx, req.Storage, func(ctx context.Context, storage logical.Storage) error {
		err := b.periodicTask(ctx, storage, config, publisherRepository)
		if err != nil {
			b.Logger().Error(fmt.Sprintf("Periodic task failed: %s", err))
		} else {
			b.Logger().Info("Periodic task succeeded")
		}
		return err
	})

	if err == tasks_manager.ErrBusy {
		b.Logger().Debug(fmt.Sprintf("Will not add new periodic task: there is currently running task which took more than %s", periodicRunPeriod))
		return nil
	}

	if err != nil {
		return fmt.Errorf("unable to add queue manager periodic task: %w", err)
	}

	if err := req.Storage.Put(ctx, &logical.StorageEntry{Key: lastPeriodicRunTimestampKey, Value: []byte(fmt.Sprintf("%d", now.Unix()))}); err != nil {
		return fmt.Errorf("unable to put last periodic task run timestamp in storage by key %q: %w", lastPeriodicRunTimestampKey, err)
	}

	b.Logger().Debug(fmt.Sprintf("Added new periodic task with uuid %s", uuid))

	return nil
}

func (b *Backend) periodicTask(ctx context.Context, storage logical.Storage, _ *configuration, publisherRepository publisher.RepositoryInterface) error {
	logboek.Context(ctx).Default().LogF("Started TUF repository keys rotation\n")
	b.Logger().Debug("Started TUF repository keys rotation")

	if err := b.Publisher.RotateRepositoryKeys(ctx, storage, publisherRepository, SystemClock); err != nil {
		return fmt.Errorf("unable to rotate TUF repository private keys: %w", err)
	}

	logboek.Context(ctx).Default().LogF("Started TUF repository timestamps update\n")
	b.Logger().Debug("Started TUF repository timestamps update")

	if err := b.Publisher.UpdateTimestamps(ctx, storage, publisherRepository, SystemClock); err != nil {
		return fmt.Errorf("unable to update TUF repository timestamps: %w", err)
	}

	return nil
}
