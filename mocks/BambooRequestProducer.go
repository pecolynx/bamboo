// Code generated by mockery v2.34.2. DO NOT EDIT.

package bamboo

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// BambooRequestProducer is an autogenerated mock type for the BambooRequestProducer type
type BambooRequestProducer struct {
	mock.Mock
}

type BambooRequestProducer_Expecter struct {
	mock *mock.Mock
}

func (_m *BambooRequestProducer) EXPECT() *BambooRequestProducer_Expecter {
	return &BambooRequestProducer_Expecter{mock: &_m.Mock}
}

// Close provides a mock function with given fields: ctx
func (_m *BambooRequestProducer) Close(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// BambooRequestProducer_Close_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Close'
type BambooRequestProducer_Close_Call struct {
	*mock.Call
}

// Close is a helper method to define mock.On call
//   - ctx context.Context
func (_e *BambooRequestProducer_Expecter) Close(ctx interface{}) *BambooRequestProducer_Close_Call {
	return &BambooRequestProducer_Close_Call{Call: _e.mock.On("Close", ctx)}
}

func (_c *BambooRequestProducer_Close_Call) Run(run func(ctx context.Context)) *BambooRequestProducer_Close_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *BambooRequestProducer_Close_Call) Return(_a0 error) *BambooRequestProducer_Close_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *BambooRequestProducer_Close_Call) RunAndReturn(run func(context.Context) error) *BambooRequestProducer_Close_Call {
	_c.Call.Return(run)
	return _c
}

// Produce provides a mock function with given fields: ctx, resultChannel, heartbeatIntervalSec, jobTimeoutSec, headers, data
func (_m *BambooRequestProducer) Produce(ctx context.Context, resultChannel string, heartbeatIntervalSec int, jobTimeoutSec int, headers map[string]string, data []byte) error {
	ret := _m.Called(ctx, resultChannel, heartbeatIntervalSec, jobTimeoutSec, headers, data)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, int, int, map[string]string, []byte) error); ok {
		r0 = rf(ctx, resultChannel, heartbeatIntervalSec, jobTimeoutSec, headers, data)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// BambooRequestProducer_Produce_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Produce'
type BambooRequestProducer_Produce_Call struct {
	*mock.Call
}

// Produce is a helper method to define mock.On call
//   - ctx context.Context
//   - resultChannel string
//   - heartbeatIntervalSec int
//   - jobTimeoutSec int
//   - headers map[string]string
//   - data []byte
func (_e *BambooRequestProducer_Expecter) Produce(ctx interface{}, resultChannel interface{}, heartbeatIntervalSec interface{}, jobTimeoutSec interface{}, headers interface{}, data interface{}) *BambooRequestProducer_Produce_Call {
	return &BambooRequestProducer_Produce_Call{Call: _e.mock.On("Produce", ctx, resultChannel, heartbeatIntervalSec, jobTimeoutSec, headers, data)}
}

func (_c *BambooRequestProducer_Produce_Call) Run(run func(ctx context.Context, resultChannel string, heartbeatIntervalSec int, jobTimeoutSec int, headers map[string]string, data []byte)) *BambooRequestProducer_Produce_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(int), args[3].(int), args[4].(map[string]string), args[5].([]byte))
	})
	return _c
}

func (_c *BambooRequestProducer_Produce_Call) Return(_a0 error) *BambooRequestProducer_Produce_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *BambooRequestProducer_Produce_Call) RunAndReturn(run func(context.Context, string, int, int, map[string]string, []byte) error) *BambooRequestProducer_Produce_Call {
	_c.Call.Return(run)
	return _c
}

// NewBambooRequestProducer creates a new instance of BambooRequestProducer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewBambooRequestProducer(t interface {
	mock.TestingT
	Cleanup(func())
}) *BambooRequestProducer {
	mock := &BambooRequestProducer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
