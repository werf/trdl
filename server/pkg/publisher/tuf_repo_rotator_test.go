package publisher

import (
	"time"

	"github.com/hashicorp/go-hclog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("TUF repo rotator", func() {
	It("should rotate roles based on expiration timestamps", func() {
		var prevRootExpires, prevTargetsExpires, prevSnapshotExpires, prevTimestampExpires time.Time

		now := time.Now()

		testRepo := &testTufRepoRotatorAccessor{
			rootExpires:      now,
			targetsExpires:   now,
			snapshotExpires:  now,
			timestampExpires: now,
		}

		rotator := NewTufRepoRotator(testRepo)

		Expect(rotator.Rotate(hclog.Default(), now)).To(Succeed())
		Expect(testRepo.rootExpires).To(Equal(now.AddDate(1, 0, 0)))
		Expect(testRepo.targetsExpires).To(Equal(now.AddDate(0, 3, 0)))
		Expect(testRepo.snapshotExpires).To(Equal(now.AddDate(0, 0, 7)))
		Expect(testRepo.timestampExpires).To(Equal(now.AddDate(0, 0, 1)))
		prevRootExpires = testRepo.rootExpires
		prevTargetsExpires = testRepo.targetsExpires
		prevSnapshotExpires = testRepo.snapshotExpires
		prevTimestampExpires = testRepo.timestampExpires

		By("passed 2 hours")
		now = now.Add(2 * time.Hour)
		Expect(rotator.Rotate(hclog.Default(), now)).To(Succeed())
		Expect(testRepo.rootExpires).To(Equal(prevRootExpires))
		Expect(testRepo.targetsExpires).To(Equal(prevTargetsExpires))
		Expect(testRepo.snapshotExpires).To(Equal(prevSnapshotExpires))
		Expect(testRepo.timestampExpires).To(Equal(prevTimestampExpires))
		prevRootExpires = testRepo.rootExpires
		prevTargetsExpires = testRepo.targetsExpires
		prevSnapshotExpires = testRepo.snapshotExpires
		prevTimestampExpires = testRepo.timestampExpires
		_ = prevTimestampExpires

		By("passed 5 hours")
		now = now.Add(3 * time.Hour)
		Expect(rotator.Rotate(hclog.Default(), now)).To(Succeed())
		Expect(testRepo.rootExpires).To(Equal(prevRootExpires))
		Expect(testRepo.targetsExpires).To(Equal(prevTargetsExpires))
		Expect(testRepo.snapshotExpires).To(Equal(prevSnapshotExpires))
		// rotate every 4th hour
		Expect(testRepo.timestampExpires).To(Equal(now.AddDate(0, 0, 1)))
		prevRootExpires = testRepo.rootExpires
		prevTargetsExpires = testRepo.targetsExpires
		prevSnapshotExpires = testRepo.snapshotExpires
		prevTimestampExpires = testRepo.timestampExpires
		_ = prevSnapshotExpires
		_ = prevTimestampExpires

		By("passed 1 day and 5 hours")
		now = now.AddDate(0, 0, 1)
		Expect(rotator.Rotate(hclog.Default(), now)).To(Succeed())
		Expect(testRepo.rootExpires).To(Equal(prevRootExpires))
		Expect(testRepo.targetsExpires).To(Equal(prevTargetsExpires))
		Expect(testRepo.snapshotExpires).To(Equal(prevSnapshotExpires))
		// rotate every 4th hour
		Expect(testRepo.timestampExpires).To(Equal(now.AddDate(0, 0, 1)))
		prevRootExpires = testRepo.rootExpires
		prevTargetsExpires = testRepo.targetsExpires
		prevSnapshotExpires = testRepo.snapshotExpires
		prevTimestampExpires = testRepo.timestampExpires
		_ = prevSnapshotExpires
		_ = prevTimestampExpires

		By("passed 2 days and 5 hours")
		now = now.AddDate(0, 0, 1)
		Expect(rotator.Rotate(hclog.Default(), now)).To(Succeed())
		Expect(testRepo.rootExpires).To(Equal(prevRootExpires))
		Expect(testRepo.targetsExpires).To(Equal(prevTargetsExpires))
		// rotate every 2nd day
		Expect(testRepo.snapshotExpires).To(Equal(now.AddDate(0, 0, 7)))
		// rotate every 4th hour
		Expect(testRepo.timestampExpires).To(Equal(now.AddDate(0, 0, 1)))
		prevRootExpires = testRepo.rootExpires
		prevTargetsExpires = testRepo.targetsExpires
		prevSnapshotExpires = testRepo.snapshotExpires
		prevTimestampExpires = testRepo.timestampExpires
		_ = prevTimestampExpires

		By("passed 3 days and 5 hours")
		now = now.AddDate(0, 0, 1)
		Expect(rotator.Rotate(hclog.Default(), now)).To(Succeed())
		Expect(testRepo.rootExpires).To(Equal(prevRootExpires))
		Expect(testRepo.targetsExpires).To(Equal(prevTargetsExpires))
		Expect(testRepo.snapshotExpires).To(Equal(prevSnapshotExpires))
		// rotate every 4th hour
		Expect(testRepo.timestampExpires).To(Equal(now.AddDate(0, 0, 1)))
		prevRootExpires = testRepo.rootExpires
		prevTargetsExpires = testRepo.targetsExpires
		prevSnapshotExpires = testRepo.snapshotExpires
		prevTimestampExpires = testRepo.timestampExpires
		_ = prevSnapshotExpires
		_ = prevTimestampExpires

		By("passed 20 days and 5 hours")
		now = now.AddDate(0, 0, 17)
		Expect(rotator.Rotate(hclog.Default(), now)).To(Succeed())
		Expect(testRepo.rootExpires).To(Equal(prevRootExpires))
		Expect(testRepo.targetsExpires).To(Equal(prevTargetsExpires))
		// rotate every 2nd day
		Expect(testRepo.snapshotExpires).To(Equal(now.AddDate(0, 0, 7)))
		// rotate every 4th hour
		Expect(testRepo.timestampExpires).To(Equal(now.AddDate(0, 0, 1)))
		prevRootExpires = testRepo.rootExpires
		prevTargetsExpires = testRepo.targetsExpires
		prevSnapshotExpires = testRepo.snapshotExpires
		prevTimestampExpires = testRepo.timestampExpires
		_ = prevTargetsExpires
		_ = prevSnapshotExpires
		_ = prevTimestampExpires

		By("passed 21 days and 5 hours")
		now = now.AddDate(0, 0, 1)
		Expect(rotator.Rotate(hclog.Default(), now)).To(Succeed())
		Expect(testRepo.rootExpires).To(Equal(prevRootExpires))
		// rotate every 21st day (3 weeks)
		Expect(testRepo.targetsExpires).To(Equal(now.AddDate(0, 3, 0)))
		// rotated not because of rotation period, but as dependant role of targets
		Expect(testRepo.snapshotExpires).To(Equal(now.AddDate(0, 0, 7)))
		// rotate every 4th hour
		Expect(testRepo.timestampExpires).To(Equal(now.AddDate(0, 0, 1)))
		prevRootExpires = testRepo.rootExpires
		prevTargetsExpires = testRepo.targetsExpires
		prevSnapshotExpires = testRepo.snapshotExpires
		prevTimestampExpires = testRepo.timestampExpires
		_ = prevTargetsExpires
		_ = prevSnapshotExpires
		_ = prevTimestampExpires

		By("passed 2 months, 21 days and 5 hours")
		now = now.AddDate(0, 2, 0)
		Expect(rotator.Rotate(hclog.Default(), now)).To(Succeed())
		Expect(testRepo.rootExpires).To(Equal(prevRootExpires))
		// rotate every 21st day (3 weeks)
		Expect(testRepo.targetsExpires).To(Equal(now.AddDate(0, 3, 0)))
		// rotate every 2nd day
		Expect(testRepo.snapshotExpires).To(Equal(now.AddDate(0, 0, 7)))
		// rotate every 4th hour
		Expect(testRepo.timestampExpires).To(Equal(now.AddDate(0, 0, 1)))
		prevRootExpires = testRepo.rootExpires
		prevTargetsExpires = testRepo.targetsExpires
		prevSnapshotExpires = testRepo.snapshotExpires
		prevTimestampExpires = testRepo.timestampExpires
		_ = prevRootExpires
		_ = prevTargetsExpires
		_ = prevSnapshotExpires
		_ = prevTimestampExpires

		By("passed 3 months, 21 days and 5 hours")
		now = now.AddDate(0, 1, 0)
		Expect(rotator.Rotate(hclog.Default(), now)).To(Succeed())
		// rotate every 3rd month-
		Expect(testRepo.rootExpires).To(Equal(now.AddDate(1, 0, 0)))
		// rotate every 21st day (3 weeks)
		Expect(testRepo.targetsExpires).To(Equal(now.AddDate(0, 3, 0)))
		// rotate every 2nd day
		Expect(testRepo.snapshotExpires).To(Equal(now.AddDate(0, 0, 7)))
		// rotate every 4th hour
		Expect(testRepo.timestampExpires).To(Equal(now.AddDate(0, 0, 1)))
		prevRootExpires = testRepo.rootExpires
		prevTargetsExpires = testRepo.targetsExpires
		prevSnapshotExpires = testRepo.snapshotExpires
		prevTimestampExpires = testRepo.timestampExpires
		_ = prevRootExpires
		_ = prevTargetsExpires
		_ = prevSnapshotExpires
		_ = prevTimestampExpires

		By("passed 1 year, 3 months, 21 days and 5 hours")
		now = now.AddDate(1, 0, 0)
		Expect(rotator.Rotate(hclog.Default(), now)).To(Succeed())
		// rotate every 3rd month
		Expect(testRepo.rootExpires).To(Equal(now.AddDate(1, 0, 0)))
		// rotate every 21st day (3 weeks)
		Expect(testRepo.targetsExpires).To(Equal(now.AddDate(0, 3, 0)))
		// rotate every 2nd day
		Expect(testRepo.snapshotExpires).To(Equal(now.AddDate(0, 0, 7)))
		// rotate every 4th hour
		Expect(testRepo.timestampExpires).To(Equal(now.AddDate(0, 0, 1)))
		prevRootExpires = testRepo.rootExpires
		prevTargetsExpires = testRepo.targetsExpires
		prevSnapshotExpires = testRepo.snapshotExpires
		prevTimestampExpires = testRepo.timestampExpires

		By("passed 1 year, 3 months, 21 days and 6 hours")
		now = now.Add(1 * time.Hour)
		Expect(rotator.Rotate(hclog.Default(), now)).To(Succeed())
		Expect(testRepo.rootExpires).To(Equal(prevRootExpires))
		Expect(testRepo.targetsExpires).To(Equal(prevTargetsExpires))
		Expect(testRepo.snapshotExpires).To(Equal(prevSnapshotExpires))
		Expect(testRepo.timestampExpires).To(Equal(prevTimestampExpires))
		prevRootExpires = testRepo.rootExpires
		prevTargetsExpires = testRepo.targetsExpires
		prevSnapshotExpires = testRepo.snapshotExpires
		prevTimestampExpires = testRepo.timestampExpires
		_ = prevRootExpires
		_ = prevTargetsExpires
		_ = prevSnapshotExpires
		_ = prevTimestampExpires

		By("passed 1 year, 7 months, 21 days and 6 hours")
		now = now.AddDate(0, 4, 0)
		Expect(rotator.Rotate(hclog.Default(), now)).To(Succeed())
		// rotate every 3rd month-
		Expect(testRepo.rootExpires).To(Equal(now.AddDate(1, 0, 0)))
		// rotate every 21st day (3 weeks)
		Expect(testRepo.targetsExpires).To(Equal(now.AddDate(0, 3, 0)))
		// rotate every 2nd day
		Expect(testRepo.snapshotExpires).To(Equal(now.AddDate(0, 0, 7)))
		// rotate every 4th hour
		Expect(testRepo.timestampExpires).To(Equal(now.AddDate(0, 0, 1)))
		prevRootExpires = testRepo.rootExpires
		prevTargetsExpires = testRepo.targetsExpires
		prevSnapshotExpires = testRepo.snapshotExpires
		prevTimestampExpires = testRepo.timestampExpires
		_ = prevRootExpires
		_ = prevTargetsExpires
		_ = prevSnapshotExpires
		_ = prevTimestampExpires
	})
})

type testTufRepoRotatorAccessor struct {
	rootExpires      time.Time
	targetsExpires   time.Time
	snapshotExpires  time.Time
	timestampExpires time.Time
}

func (repo *testTufRepoRotatorAccessor) RootExpires() (time.Time, error) {
	return repo.rootExpires, nil
}

func (repo *testTufRepoRotatorAccessor) TargetsExpires() (time.Time, error) {
	return repo.targetsExpires, nil
}

func (repo *testTufRepoRotatorAccessor) SnapshotExpires() (time.Time, error) {
	return repo.snapshotExpires, nil
}

func (repo *testTufRepoRotatorAccessor) TimestampExpires() (time.Time, error) {
	return repo.timestampExpires, nil
}

func (repo *testTufRepoRotatorAccessor) IncrementRootVersionWithExpires(expires time.Time) error {
	repo.rootExpires = expires
	return nil
}

func (repo *testTufRepoRotatorAccessor) IncrementTargetsVersionWithExpires(expires time.Time) error {
	repo.targetsExpires = expires
	return nil
}

func (repo *testTufRepoRotatorAccessor) IncrementSnapshotVersionWithExpires(expires time.Time) error {
	repo.snapshotExpires = expires
	return nil
}

func (repo *testTufRepoRotatorAccessor) IncrementTimestampVersionWithExpires(expires time.Time) error {
	repo.timestampExpires = expires
	return nil
}

func (repo *testTufRepoRotatorAccessor) Commit() error {
	return nil
}
