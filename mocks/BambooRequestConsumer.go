// Code generated by mockery v2.34.2. DO NOT EDIT.

package bamboo

import (
	context "context"

	proto "github.com/pecolynx/bamboo/proto"
	mock "github.com/stretchr/testify/mock"
)

// BambooRequestConsumer is an autogenerated mock type for the BambooRequestConsumer type
type BambooRequestConsumer struct {
	mock.Mock
}

type BambooRequestConsumer_Expecter struct {
	mock *mock.Mock
}

func (_m *BambooRequestConsumer) EXPECT() *BambooRequestConsumer_Expecter {
	return &BambooRequestConsumer_Expecter{mock: &_m.Mock}
}

// Close provides a mock function with given fields: ctx
func (_m *BambooRequestConsumer) Close(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// BambooRequestConsumer_Close_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Close'
type BambooRequestConsumer_Close_Call struct {
	*mock.Call
}

// Close is a helper method to define mock.On call
//   - ctx context.Context
func (_e *BambooRequestConsumer_Expecter) Close(ctx interface{}) *BambooRequestConsumer_Close_Call {
	return &BambooRequestConsumer_Close_Call{Call: _e.mock.On("Close", ctx)}
}

func (_c *BambooRequestConsumer_Close_Call) Run(run func(ctx context.Context)) *BambooRequestConsumer_Close_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *BambooRequestConsumer_Close_Call) Return(_a0 error) *BambooRequestConsumer_Close_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *BambooRequestConsumer_Close_Call) RunAndReturn(run func(context.Context) error) *BambooRequestConsumer_Close_Call {
	_c.Call.Return(run)
	return _c
}

// Consume provides a mock function with given fields: ctx
func (_m *BambooRequestConsumer) Consume(ctx context.Context) (*proto.WorkerParameter, error) {
	ret := _m.Called(ctx)

	var r0 *proto.WorkerParameter
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (*proto.WorkerParameter, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) *proto.WorkerParameter); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*proto.WorkerParameter)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// BambooRequestConsumer_Consume_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Consume'
type BambooRequestConsumer_Consume_Call struct {
	*mock.Call
}

// Consume is a helper method to define mock.On call
//   - ctx context.Context
func (_e *BambooRequestConsumer_Expecter) Consume(ctx interface{}) *BambooRequestConsumer_Consume_Call {
	return &BambooRequestConsumer_Consume_Call{Call: _e.mock.On("Consume", ctx)}
}

func (_c *BambooRequestConsumer_Consume_Call) Run(run func(ctx context.Context)) *BambooRequestConsumer_Consume_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *BambooRequestConsumer_Consume_Call) Return(_a0 *proto.WorkerParameter, _a1 error) *BambooRequestConsumer_Consume_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *BambooRequestConsumer_Consume_Call) RunAndReturn(run func(context.Context) (*proto.WorkerParameter, error)) *BambooRequestConsumer_Consume_Call {
	_c.Call.Return(run)
	return _c
}

// Ping provides a mock function with given fields: ctx
func (_m *BambooRequestConsumer) Ping(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// BambooRequestConsumer_Ping_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Ping'
type BambooRequestConsumer_Ping_Call struct {
	*mock.Call
}

// Ping is a helper method to define mock.On call
//   - ctx context.Context
func (_e *BambooRequestConsumer_Expecter) Ping(ctx interface{}) *BambooRequestConsumer_Ping_Call {
	return &BambooRequestConsumer_Ping_Call{Call: _e.mock.On("Ping", ctx)}
}

func (_c *BambooRequestConsumer_Ping_Call) Run(run func(ctx context.Context)) *BambooRequestConsumer_Ping_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *BambooRequestConsumer_Ping_Call) Return(_a0 error) *BambooRequestConsumer_Ping_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *BambooRequestConsumer_Ping_Call) RunAndReturn(run func(context.Context) error) *BambooRequestConsumer_Ping_Call {
	_c.Call.Return(run)
	return _c
}

// NewBambooRequestConsumer creates a new instance of BambooRequestConsumer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewBambooRequestConsumer(t interface {
	mock.TestingT
	Cleanup(func())
}) *BambooRequestConsumer {
	mock := &BambooRequestConsumer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
