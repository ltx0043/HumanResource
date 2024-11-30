// Code generated by mockery v2.46.3. DO NOT EDIT.

package mocks

import (
	context "context"

	logpoller "github.com/smartcontractkit/chainlink/v2/core/chains/evm/logpoller"
	mock "github.com/stretchr/testify/mock"
)

// Registrar is an autogenerated mock type for the Registrar type
type Registrar struct {
	mock.Mock
}

type Registrar_Expecter struct {
	mock *mock.Mock
}

func (_m *Registrar) EXPECT() *Registrar_Expecter {
	return &Registrar_Expecter{mock: &_m.Mock}
}

// HasFilter provides a mock function with given fields: _a0
func (_m *Registrar) HasFilter(_a0 string) bool {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for HasFilter")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Registrar_HasFilter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'HasFilter'
type Registrar_HasFilter_Call struct {
	*mock.Call
}

// HasFilter is a helper method to define mock.On call
//   - _a0 string
func (_e *Registrar_Expecter) HasFilter(_a0 interface{}) *Registrar_HasFilter_Call {
	return &Registrar_HasFilter_Call{Call: _e.mock.On("HasFilter", _a0)}
}

func (_c *Registrar_HasFilter_Call) Run(run func(_a0 string)) *Registrar_HasFilter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *Registrar_HasFilter_Call) Return(_a0 bool) *Registrar_HasFilter_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Registrar_HasFilter_Call) RunAndReturn(run func(string) bool) *Registrar_HasFilter_Call {
	_c.Call.Return(run)
	return _c
}

// RegisterFilter provides a mock function with given fields: _a0, _a1
func (_m *Registrar) RegisterFilter(_a0 context.Context, _a1 logpoller.Filter) error {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for RegisterFilter")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, logpoller.Filter) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Registrar_RegisterFilter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RegisterFilter'
type Registrar_RegisterFilter_Call struct {
	*mock.Call
}

// RegisterFilter is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 logpoller.Filter
func (_e *Registrar_Expecter) RegisterFilter(_a0 interface{}, _a1 interface{}) *Registrar_RegisterFilter_Call {
	return &Registrar_RegisterFilter_Call{Call: _e.mock.On("RegisterFilter", _a0, _a1)}
}

func (_c *Registrar_RegisterFilter_Call) Run(run func(_a0 context.Context, _a1 logpoller.Filter)) *Registrar_RegisterFilter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(logpoller.Filter))
	})
	return _c
}

func (_c *Registrar_RegisterFilter_Call) Return(_a0 error) *Registrar_RegisterFilter_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Registrar_RegisterFilter_Call) RunAndReturn(run func(context.Context, logpoller.Filter) error) *Registrar_RegisterFilter_Call {
	_c.Call.Return(run)
	return _c
}

// UnregisterFilter provides a mock function with given fields: _a0, _a1
func (_m *Registrar) UnregisterFilter(_a0 context.Context, _a1 string) error {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for UnregisterFilter")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Registrar_UnregisterFilter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UnregisterFilter'
type Registrar_UnregisterFilter_Call struct {
	*mock.Call
}

// UnregisterFilter is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 string
func (_e *Registrar_Expecter) UnregisterFilter(_a0 interface{}, _a1 interface{}) *Registrar_UnregisterFilter_Call {
	return &Registrar_UnregisterFilter_Call{Call: _e.mock.On("UnregisterFilter", _a0, _a1)}
}

func (_c *Registrar_UnregisterFilter_Call) Run(run func(_a0 context.Context, _a1 string)) *Registrar_UnregisterFilter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *Registrar_UnregisterFilter_Call) Return(_a0 error) *Registrar_UnregisterFilter_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Registrar_UnregisterFilter_Call) RunAndReturn(run func(context.Context, string) error) *Registrar_UnregisterFilter_Call {
	_c.Call.Return(run)
	return _c
}

// NewRegistrar creates a new instance of Registrar. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewRegistrar(t interface {
	mock.TestingT
	Cleanup(func())
}) *Registrar {
	mock := &Registrar{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
