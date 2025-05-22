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

	should := require.New(ts.T())
	should.Zero(ts.tw.GetTaskSize())

	ts.tw.Stop()
}

func (ts *TimeWheelTestSuite) TestRunTask() {
	should := require.New(ts.T())
	should.Zero(ts.tw.GetTaskSize())

	ts.tw.AddTask(-time.Millisecond*10, "-10ms") // will be ignored
	ts.tw.AddTask(time.Millisecond*10, "10ms")   // run immediately

	should.Zero(ts.tw.GetTaskSize())

	ts.tw.AddTask(time.Millisecond*300, "300ms")
	ts.tw.AddTask(time.Millisecond*100, "100ms")
	k := ts.tw.AddTask(time.Millisecond*100, "100ms")
	ts.tw.AddTask(time.Millisecond*800, "800ms")
	ts.tw.AddTask(time.Second, "1s")
	ts.tw.AddTask(time.Second*2, "2s")
	ts.tw.AddTask(time.Millisecond*1500, "1.5s")

	should.Equal(7, ts.tw.GetTaskSize())

	ts.tw.RemoveTask(k)
	ts.tw.RemoveTask("fake-key") // will be ignored because it is not in the task list

	should.Equal(6, ts.tw.GetTaskSize())
}
