// Code generated by mockery v2.34.2. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// BambooHeartbeatPublisher is an autogenerated mock type for the BambooHeartbeatPublisher type
type BambooHeartbeatPublisher struct {
	mock.Mock
}

type BambooHeartbeatPublisher_Expecter struct {
	mock *mock.Mock
}

func (_m *BambooHeartbeatPublisher) EXPECT() *BambooHeartbeatPublisher_Expecter {
	return &BambooHeartbeatPublisher_Expecter{mock: &_m.Mock}
}

// Ping provides a mock function with given fields: ctx
func (_m *BambooHeartbeatPublisher) Ping(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// BambooHeartbeatPublisher_Ping_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Ping'
type BambooHeartbeatPublisher_Ping_Call struct {
	*mock.Call
}

// Ping is a helper method to define mock.On call
//   - ctx context.Context
func (_e *BambooHeartbeatPublisher_Expecter) Ping(ctx interface{}) *BambooHeartbeatPublisher_Ping_Call {
	return &BambooHeartbeatPublisher_Ping_Call{Call: _e.mock.On("Ping", ctx)}
}

func (_c *BambooHeartbeatPublisher_Ping_Call) Run(run func(ctx context.Context)) *BambooHeartbeatPublisher_Ping_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *BambooHeartbeatPublisher_Ping_Call) Return(_a0 error) *BambooHeartbeatPublisher_Ping_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *BambooHeartbeatPublisher_Ping_Call) RunAndReturn(run func(context.Context) error) *BambooHeartbeatPublisher_Ping_Call {
	_c.Call.Return(run)
	return _c
}

// Run provides a mock function with given fields: ctx, resultChannel, heartbeatIntervalSec, done, aborted
func (_m *BambooHeartbeatPublisher) Run(ctx context.Context, resultChannel string, heartbeatIntervalSec int, done <-chan interface{}, aborted <-chan interface{}) {
	_m.Called(ctx, resultChannel, heartbeatIntervalSec, done, aborted)
}

// BambooHeartbeatPublisher_Run_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Run'
type BambooHeartbeatPublisher_Run_Call struct {
	*mock.Call
}

// Run is a helper method to define mock.On call
//   - ctx context.Context
//   - resultChannel string
//   - heartbeatIntervalSec int
//   - done <-chan interface{}
//   - aborted <-chan interface{}
func (_e *BambooHeartbeatPublisher_Expecter) Run(ctx interface{}, resultChannel interface{}, heartbeatIntervalSec interface{}, done interface{}, aborted interface{}) *BambooHeartbeatPublisher_Run_Call {
	return &BambooHeartbeatPublisher_Run_Call{Call: _e.mock.On("Run", ctx, resultChannel, heartbeatIntervalSec, done, aborted)}
}

func (_c *BambooHeartbeatPublisher_Run_Call) Run(run func(ctx context.Context, resultChannel string, heartbeatIntervalSec int, done <-chan interface{}, aborted <-chan interface{})) *BambooHeartbeatPublisher_Run_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(int), args[3].(<-chan interface{}), args[4].(<-chan interface{}))
	})
	return _c
}

func (_c *BambooHeartbeatPublisher_Run_Call) Return() *BambooHeartbeatPublisher_Run_Call {
	_c.Call.Return()
	return _c
}

func (_c *BambooHeartbeatPublisher_Run_Call) RunAndReturn(run func(context.Context, string, int, <-chan interface{}, <-chan interface{})) *BambooHeartbeatPublisher_Run_Call {
	_c.Call.Return(run)
	return _c
}

// NewBambooHeartbeatPublisher creates a new instance of BambooHeartbeatPublisher. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewBambooHeartbeatPublisher(t interface {
	mock.TestingT
	Cleanup(func())
}) *BambooHeartbeatPublisher {
	mock := &BambooHeartbeatPublisher{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
