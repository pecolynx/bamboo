// Code generated by mockery v2.34.2. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// BambooResultSubscriber is an autogenerated mock type for the BambooResultSubscriber type
type BambooResultSubscriber struct {
	mock.Mock
}

type BambooResultSubscriber_Expecter struct {
	mock *mock.Mock
}

func (_m *BambooResultSubscriber) EXPECT() *BambooResultSubscriber_Expecter {
	return &BambooResultSubscriber_Expecter{mock: &_m.Mock}
}

// Ping provides a mock function with given fields: ctx
func (_m *BambooResultSubscriber) Ping(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// BambooResultSubscriber_Ping_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Ping'
type BambooResultSubscriber_Ping_Call struct {
	*mock.Call
}

// Ping is a helper method to define mock.On call
//   - ctx context.Context
func (_e *BambooResultSubscriber_Expecter) Ping(ctx interface{}) *BambooResultSubscriber_Ping_Call {
	return &BambooResultSubscriber_Ping_Call{Call: _e.mock.On("Ping", ctx)}
}

func (_c *BambooResultSubscriber_Ping_Call) Run(run func(ctx context.Context)) *BambooResultSubscriber_Ping_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *BambooResultSubscriber_Ping_Call) Return(_a0 error) *BambooResultSubscriber_Ping_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *BambooResultSubscriber_Ping_Call) RunAndReturn(run func(context.Context) error) *BambooResultSubscriber_Ping_Call {
	_c.Call.Return(run)
	return _c
}

// Subscribe provides a mock function with given fields: ctx, resultChannel, heartbeatIntervalSec, jobTimeoutSec
func (_m *BambooResultSubscriber) Subscribe(ctx context.Context, resultChannel string, heartbeatIntervalSec int, jobTimeoutSec int) ([]byte, error) {
	ret := _m.Called(ctx, resultChannel, heartbeatIntervalSec, jobTimeoutSec)

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, int, int) ([]byte, error)); ok {
		return rf(ctx, resultChannel, heartbeatIntervalSec, jobTimeoutSec)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, int, int) []byte); ok {
		r0 = rf(ctx, resultChannel, heartbeatIntervalSec, jobTimeoutSec)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, int, int) error); ok {
		r1 = rf(ctx, resultChannel, heartbeatIntervalSec, jobTimeoutSec)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// BambooResultSubscriber_Subscribe_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Subscribe'
type BambooResultSubscriber_Subscribe_Call struct {
	*mock.Call
}

// Subscribe is a helper method to define mock.On call
//   - ctx context.Context
//   - resultChannel string
//   - heartbeatIntervalSec int
//   - jobTimeoutSec int
func (_e *BambooResultSubscriber_Expecter) Subscribe(ctx interface{}, resultChannel interface{}, heartbeatIntervalSec interface{}, jobTimeoutSec interface{}) *BambooResultSubscriber_Subscribe_Call {
	return &BambooResultSubscriber_Subscribe_Call{Call: _e.mock.On("Subscribe", ctx, resultChannel, heartbeatIntervalSec, jobTimeoutSec)}
}

func (_c *BambooResultSubscriber_Subscribe_Call) Run(run func(ctx context.Context, resultChannel string, heartbeatIntervalSec int, jobTimeoutSec int)) *BambooResultSubscriber_Subscribe_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(int), args[3].(int))
	})
	return _c
}

func (_c *BambooResultSubscriber_Subscribe_Call) Return(_a0 []byte, _a1 error) *BambooResultSubscriber_Subscribe_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *BambooResultSubscriber_Subscribe_Call) RunAndReturn(run func(context.Context, string, int, int) ([]byte, error)) *BambooResultSubscriber_Subscribe_Call {
	_c.Call.Return(run)
	return _c
}

// NewBambooResultSubscriber creates a new instance of BambooResultSubscriber. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewBambooResultSubscriber(t interface {
	mock.TestingT
	Cleanup(func())
}) *BambooResultSubscriber {
	mock := &BambooResultSubscriber{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}