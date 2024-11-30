// Code generated by mockery v2.46.3. DO NOT EDIT.

package mocks

import (
	pkgservices "github.com/smartcontractkit/chainlink-common/pkg/services"
	mock "github.com/stretchr/testify/mock"
)

// Checker is an autogenerated mock type for the Checker type
type Checker struct {
	mock.Mock
}

type Checker_Expecter struct {
	mock *mock.Mock
}

func (_m *Checker) EXPECT() *Checker_Expecter {
	return &Checker_Expecter{mock: &_m.Mock}
}

// Close provides a mock function with given fields:
func (_m *Checker) Close() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Close")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Checker_Close_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Close'
type Checker_Close_Call struct {
	*mock.Call
}

// Close is a helper method to define mock.On call
func (_e *Checker_Expecter) Close() *Checker_Close_Call {
	return &Checker_Close_Call{Call: _e.mock.On("Close")}
}

func (_c *Checker_Close_Call) Run(run func()) *Checker_Close_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Checker_Close_Call) Return(_a0 error) *Checker_Close_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Checker_Close_Call) RunAndReturn(run func() error) *Checker_Close_Call {
	_c.Call.Return(run)
	return _c
}

// IsHealthy provides a mock function with given fields:
func (_m *Checker) IsHealthy() (bool, map[string]error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for IsHealthy")
	}

	var r0 bool
	var r1 map[string]error
	if rf, ok := ret.Get(0).(func() (bool, map[string]error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func() map[string]error); ok {
		r1 = rf()
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(map[string]error)
		}
	}

	return r0, r1
}

// Checker_IsHealthy_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'IsHealthy'
type Checker_IsHealthy_Call struct {
	*mock.Call
}

// IsHealthy is a helper method to define mock.On call
func (_e *Checker_Expecter) IsHealthy() *Checker_IsHealthy_Call {
	return &Checker_IsHealthy_Call{Call: _e.mock.On("IsHealthy")}
}

func (_c *Checker_IsHealthy_Call) Run(run func()) *Checker_IsHealthy_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Checker_IsHealthy_Call) Return(healthy bool, errors map[string]error) *Checker_IsHealthy_Call {
	_c.Call.Return(healthy, errors)
	return _c
}

func (_c *Checker_IsHealthy_Call) RunAndReturn(run func() (bool, map[string]error)) *Checker_IsHealthy_Call {
	_c.Call.Return(run)
	return _c
}

// IsReady provides a mock function with given fields:
func (_m *Checker) IsReady() (bool, map[string]error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for IsReady")
	}

	var r0 bool
	var r1 map[string]error
	if rf, ok := ret.Get(0).(func() (bool, map[string]error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func() map[string]error); ok {
		r1 = rf()
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(map[string]error)
		}
	}

	return r0, r1
}

// Checker_IsReady_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'IsReady'
type Checker_IsReady_Call struct {
	*mock.Call
}

// IsReady is a helper method to define mock.On call
func (_e *Checker_Expecter) IsReady() *Checker_IsReady_Call {
	return &Checker_IsReady_Call{Call: _e.mock.On("IsReady")}
}

func (_c *Checker_IsReady_Call) Run(run func()) *Checker_IsReady_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Checker_IsReady_Call) Return(ready bool, errors map[string]error) *Checker_IsReady_Call {
	_c.Call.Return(ready, errors)
	return _c
}

func (_c *Checker_IsReady_Call) RunAndReturn(run func() (bool, map[string]error)) *Checker_IsReady_Call {
	_c.Call.Return(run)
	return _c
}

// Register provides a mock function with given fields: service
func (_m *Checker) Register(service pkgservices.HealthReporter) error {
	ret := _m.Called(service)

	if len(ret) == 0 {
		panic("no return value specified for Register")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(pkgservices.HealthReporter) error); ok {
		r0 = rf(service)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Checker_Register_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Register'
type Checker_Register_Call struct {
	*mock.Call
}

// Register is a helper method to define mock.On call
//   - service pkgservices.HealthReporter
func (_e *Checker_Expecter) Register(service interface{}) *Checker_Register_Call {
	return &Checker_Register_Call{Call: _e.mock.On("Register", service)}
}

func (_c *Checker_Register_Call) Run(run func(service pkgservices.HealthReporter)) *Checker_Register_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(pkgservices.HealthReporter))
	})
	return _c
}

func (_c *Checker_Register_Call) Return(_a0 error) *Checker_Register_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Checker_Register_Call) RunAndReturn(run func(pkgservices.HealthReporter) error) *Checker_Register_Call {
	_c.Call.Return(run)
	return _c
}

// Start provides a mock function with given fields:
func (_m *Checker) Start() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Start")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Checker_Start_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Start'
type Checker_Start_Call struct {
	*mock.Call
}

// Start is a helper method to define mock.On call
func (_e *Checker_Expecter) Start() *Checker_Start_Call {
	return &Checker_Start_Call{Call: _e.mock.On("Start")}
}

func (_c *Checker_Start_Call) Run(run func()) *Checker_Start_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Checker_Start_Call) Return(_a0 error) *Checker_Start_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Checker_Start_Call) RunAndReturn(run func() error) *Checker_Start_Call {
	_c.Call.Return(run)
	return _c
}

// Unregister provides a mock function with given fields: name
func (_m *Checker) Unregister(name string) error {
	ret := _m.Called(name)

	if len(ret) == 0 {
		panic("no return value specified for Unregister")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(name)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Checker_Unregister_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Unregister'
type Checker_Unregister_Call struct {
	*mock.Call
}

// Unregister is a helper method to define mock.On call
//   - name string
func (_e *Checker_Expecter) Unregister(name interface{}) *Checker_Unregister_Call {
	return &Checker_Unregister_Call{Call: _e.mock.On("Unregister", name)}
}

func (_c *Checker_Unregister_Call) Run(run func(name string)) *Checker_Unregister_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *Checker_Unregister_Call) Return(_a0 error) *Checker_Unregister_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Checker_Unregister_Call) RunAndReturn(run func(string) error) *Checker_Unregister_Call {
	_c.Call.Return(run)
	return _c
}

// NewChecker creates a new instance of Checker. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewChecker(t interface {
	mock.TestingT
	Cleanup(func())
}) *Checker {
	mock := &Checker{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
