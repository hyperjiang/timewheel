package timewheel_test

import (
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
	ts.tw.Stop()
}

func (ts *TimeWheelTestSuite) TestRunTask() {
	ts.tw.AddTask(-time.Millisecond*10, 0, "-10ms") // will be ignored
	ts.tw.AddTask(time.Millisecond*10, 1, "10ms")
	ts.tw.AddTask(time.Millisecond*300, 2, "300ms")
	ts.tw.AddTask(time.Millisecond*100, 3, "100ms")
	ts.tw.AddTask(time.Millisecond*100, 3, "100ms")
	ts.tw.AddTask(time.Millisecond*800, 4, "800ms")
	ts.tw.AddTask(time.Second, 5, "1s")
	ts.tw.AddTask(time.Second*2, 6, "2s")
	ts.tw.AddTask(time.Millisecond*1500, 7, "1.5s")
	ts.tw.RemoveTask(5)
	ts.tw.RemoveTask(nil)
	ts.tw.RemoveTask(100) // will be ignored because it is not in the task list
}
