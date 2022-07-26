package publisher

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
)

type TufRepoRotator struct {
	TufRepo TufRepoRotatorAccessor
}

func NewTufRepoRotator(tufRepo TufRepoRotatorAccessor) *TufRepoRotator {
	return &TufRepoRotator{TufRepo: tufRepo}
}

func (rotator *TufRepoRotator) Rotate(logger hclog.Logger, now time.Time) error {
	var changedRoot, changedTargets, changedSnapshot, changedTimestamp bool

	logger.Debug("start rotating expiration timestamps and versions of TUF repository roles")

	{
		rotateAt, err := rotator.GetRootRotateAt()
		if err != nil {
			return fmt.Errorf("unable to get root.json rotation time: %w", err)
		}
		hitRotationPeriod := rotateAt.Sub(now) <= 0

		if hitRotationPeriod {
			if err := rotator.RotateRoot(now); err != nil {
				return fmt.Errorf("unable to rotate root.json: %w", err)
			}
			changedRoot = true
			logger.Debug(fmt.Sprintf("rotated root.json TUF repository role because of hitRotationPeriod=%v\n", hitRotationPeriod))
		}
	}

	{
		rotateAt, err := rotator.GetTargetsRotateAt()
		if err != nil {
			return fmt.Errorf("unable to get targets.json rotation time: %w", err)
		}
		hitRotationPeriod := rotateAt.Sub(now) <= 0

		if hitRotationPeriod {
			if err := rotator.RotateTargets(now); err != nil {
				return fmt.Errorf("unable to rotate targets.json: %w", err)
			}
			changedTargets = true
			logger.Debug(fmt.Sprintf("rotated targets.json TUF repository role because of hitRotationPeriod=%v\n", hitRotationPeriod))
		}
	}

	{
		rotateAt, err := rotator.GetSnapshotRotateAt()
		if err != nil {
			return fmt.Errorf("unable to get snapshot.json rotation time: %w", err)
		}
		hitRotationPeriod := rotateAt.Sub(now) <= 0

		if changedRoot || changedTargets || hitRotationPeriod {
			if err := rotator.RotateSnapshot(now); err != nil {
				return fmt.Errorf("unable to rotate snapshot.json: %w", err)
			}
			changedSnapshot = true
			logger.Debug(fmt.Sprintf("rotated snapshot.json TUF repository role because of changedRoot=%v changedTargets=%v hitRotationPeriod=%v", changedRoot, changedTargets, hitRotationPeriod))
		}
	}

	{
		rotateAt, err := rotator.GetTimestampRotateAt()
		if err != nil {
			return fmt.Errorf("unable to get timestamp.json rotation time: %w", err)
		}
		hitRotationPeriod := rotateAt.Sub(now) <= 0

		if changedSnapshot || hitRotationPeriod {
			if err := rotator.RotateTimestamp(now); err != nil {
				return fmt.Errorf("unable to rotate timestamp.json: %w", err)
			}
			changedTimestamp = true
			logger.Debug(fmt.Sprintf("rotated timestamp.json TUF repository role because of changedSnapshot=%v changedTargets=%v hitRotationPeriod=%v", changedRoot, changedTargets, hitRotationPeriod))
		}

	}

	if changedRoot || changedTargets || changedSnapshot || changedTimestamp {
		logger.Debug("committing TUF repositories roles rotation ...")
		if err := rotator.Commit(); err != nil {
			return fmt.Errorf("unable to commit TUF repository roles rotation: %w", err)
		}
	}

	return nil
}

// Root expires every year, rotate every 3 month
func (rotator *TufRepoRotator) GetRootRotateAt() (time.Time, error) {
	expiresAt, err := rotator.TufRepo.RootExpires()
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to get current expires: %w", err)
	}
	return expiresAt.AddDate(-1, 0, 0).AddDate(0, 3, 0), nil
}

func (rotator *TufRepoRotator) RotateRoot(now time.Time) error {
	return rotator.TufRepo.IncrementRootVersionWithExpires(now.AddDate(1, 0, 0))
}

// Targets expires every 3 month, rotate every 3 weeks
func (rotator *TufRepoRotator) GetTargetsRotateAt() (time.Time, error) {
	expiresAt, err := rotator.TufRepo.TargetsExpires()
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to get current expires: %w", err)
	}
	return expiresAt.AddDate(0, -3, 0).AddDate(0, 0, 21), nil
}

func (rotator *TufRepoRotator) RotateTargets(now time.Time) error {
	return rotator.TufRepo.IncrementTargetsVersionWithExpires(now.AddDate(0, 3, 0))
}

// Snapshot expires every 7 days, rotate every 2nd day
func (rotator *TufRepoRotator) GetSnapshotRotateAt() (time.Time, error) {
	expiresAt, err := rotator.TufRepo.SnapshotExpires()
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to get current expires: %w", err)
	}
	return expiresAt.AddDate(0, 0, -7).AddDate(0, 0, 2), nil
}

func (rotator *TufRepoRotator) RotateSnapshot(now time.Time) error {
	return rotator.TufRepo.IncrementSnapshotVersionWithExpires(now.AddDate(0, 0, 7))
}

// Timestamp expires every day, rotate every 4th hour
func (rotator *TufRepoRotator) GetTimestampRotateAt() (time.Time, error) {
	expiresAt, err := rotator.TufRepo.TimestampExpires()
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to get current expires: %w", err)
	}
	return expiresAt.AddDate(0, 0, -1).Add(time.Hour * 4), nil
}

func (rotator *TufRepoRotator) RotateTimestamp(now time.Time) error {
	return rotator.TufRepo.IncrementTimestampVersionWithExpires(now.AddDate(0, 0, 1))
}

func (rotator *TufRepoRotator) Commit() error {
	return rotator.TufRepo.Commit()
}

type TufRepoRotatorAccessor interface {
	RootExpires() (time.Time, error)
	TargetsExpires() (time.Time, error)
	SnapshotExpires() (time.Time, error)
	TimestampExpires() (time.Time, error)

	IncrementRootVersionWithExpires(expires time.Time) error
	IncrementTargetsVersionWithExpires(expires time.Time) error
	IncrementSnapshotVersionWithExpires(expires time.Time) error
	IncrementTimestampVersionWithExpires(expires time.Time) error

	Commit() error
}
