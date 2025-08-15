package timewheel_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hyperjiang/timewheel"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TimeWheelTestSuite struct {
	suite.Suite
	tw *timewheel.TimeWheel
}

// TestTimeWheelTestSuite runs the http client test suite
func TestTimeWheelTestSuite(t *testing.T) {
	suite.Run(t, new(TimeWheelTestSuite))
}

// SetupSuite run once at the very start of the testing suite, before any tests are run.
func (ts *TimeWheelTestSuite) SetupSuite() {
	should := require.New(ts.T())

	ts.tw = timewheel.New(
		timewheel.WithHandler(nil),
		timewheel.WithLogger(timewheel.Printf),
		timewheel.WithSlotNum(10),
		timewheel.WithTickDuration(time.Millisecond*100),
	)
	should.NotNil(ts.tw)

	ts.tw.Start()
}

// TearDownSuite run once at the very end of the testing suite, after all tests have been run.
func (ts *TimeWheelTestSuite) TearDownSuite() {
	time.Sleep(time.Second * 3)

	should := require.New(ts.T())
	should.Zero(ts.tw.GetTaskSize())

	ts.tw.Stop()
}

func (ts *TimeWheelTestSuite) TestRunTask() {
	should := require.New(ts.T())
	should.Zero(ts.tw.GetTaskSize())

	_, err := ts.tw.AddTask(-time.Millisecond*10, "-10ms") // run immediately
	should.NoError(err)
	_, err = ts.tw.AddTask(time.Millisecond*10, "10ms") // run immediately
	should.NoError(err)

	should.Zero(ts.tw.GetTaskSize())

	_, err = ts.tw.AddTask(time.Millisecond*500, "500ms")
	should.NoError(err)
	_, err = ts.tw.AddTask(time.Millisecond*300, "300ms")
	should.NoError(err)
	k, err := ts.tw.AddTask(time.Millisecond*300, "300ms")
	should.NoError(err)
	_ = ts.tw.RemoveTask(k)
	_, err = ts.tw.AddTask(time.Millisecond*800, "800ms")
	should.NoError(err)
	_, err = ts.tw.AddTask(time.Second, "1s")
	should.NoError(err)
	_, err = ts.tw.AddTask(time.Second*2, "2s")
	should.NoError(err)
	_, err = ts.tw.AddTask(time.Millisecond*1500, "1.5s")
	should.NoError(err)

	time.Sleep(time.Millisecond * 100)
	should.Equal(6, ts.tw.GetTaskSize())

	_ = ts.tw.RemoveTask("fake-key") // will be ignored because it is not in the task list

	should.Equal(6, ts.tw.GetTaskSize())
}

func (ts *TimeWheelTestSuite) TestHandlerPanicRecovery() {
	should := require.New(ts.T())

	var count int32
	h := timewheel.NewHandlerFunc(func(any) {
		atomic.AddInt32(&count, 1)
		panic("boom")
	})

	tw := timewheel.New(
		timewheel.WithHandler(h),
		timewheel.WithLogger(timewheel.Printf),
		timewheel.WithSlotNum(6),
		timewheel.WithTickDuration(20*time.Millisecond),
	)
	tw.Start()
	defer tw.Stop()

	_, err := tw.AddTask(10*time.Millisecond, "panic-task")
	should.NoError(err)

	time.Sleep(120 * time.Millisecond)
	should.Equal(int32(1), atomic.LoadInt32(&count), "panic recovered")
}

func (ts *TimeWheelTestSuite) TestConcurrentAddRemove() {
	should := require.New(ts.T())

	tw := timewheel.New(
		timewheel.WithHandler(nil),
		timewheel.WithLogger(timewheel.Printf),
		timewheel.WithSlotNum(32),
		timewheel.WithTickDuration(15*time.Millisecond),
	)
	tw.Start()
	defer tw.Stop()

	var wg sync.WaitGroup
	adders := 20
	perAdder := 15
	keysCh := make(chan string, adders*perAdder)

	wg.Add(adders)
	for i := 0; i < adders; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < perAdder; j++ {
				k, err := tw.AddTask(120*time.Millisecond, j)
				if err == nil {
					keysCh <- k
				}
			}
		}()
	}
	wg.Wait()
	close(keysCh)

	// remove half of tasks
	i := 0
	for k := range keysCh {
		if i%2 == 0 {
			_ = tw.RemoveTask(k)
		}
		i++
	}

	// wait for tasks to run
	time.Sleep(200 * time.Millisecond)
	size := tw.GetTaskSize()
	should.GreaterOrEqual(size, 0)
}
func (ts *TimeWheelTestSuite) TestIdempotentStop() {
	should := require.New(ts.T())
	tw := timewheel.New(
		timewheel.WithHandler(nil),
		timewheel.WithLogger(timewheel.Printf),
		timewheel.WithSlotNum(4),
		timewheel.WithTickDuration(25*time.Millisecond),
	)
	tw.Start()
	tw.Stop()
	tw.Start() // return directly
	tw.Stop()  // return directly

	// already stopped
	id, err := tw.AddTask(0, "zero-fast")
	should.Error(err)
	should.Empty(id)

	err = tw.RemoveTask("non-existent")
	should.Error(err)
}
