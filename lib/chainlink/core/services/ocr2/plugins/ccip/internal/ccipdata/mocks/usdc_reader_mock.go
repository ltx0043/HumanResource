// Code generated by mockery v2.46.3. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// USDCReader is an autogenerated mock type for the USDCReader type
type USDCReader struct {
	mock.Mock
}

type USDCReader_Expecter struct {
	mock *mock.Mock
}

func (_m *USDCReader) EXPECT() *USDCReader_Expecter {
	return &USDCReader_Expecter{mock: &_m.Mock}
}

// GetUSDCMessagePriorToLogIndexInTx provides a mock function with given fields: ctx, logIndex, usdcTokenIndexOffset, txHash
func (_m *USDCReader) GetUSDCMessagePriorToLogIndexInTx(ctx context.Context, logIndex int64, usdcTokenIndexOffset int, txHash string) ([]byte, error) {
	ret := _m.Called(ctx, logIndex, usdcTokenIndexOffset, txHash)

	if len(ret) == 0 {
		panic("no return value specified for GetUSDCMessagePriorToLogIndexInTx")
	}

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, int64, int, string) ([]byte, error)); ok {
		return rf(ctx, logIndex, usdcTokenIndexOffset, txHash)
	}
	if rf, ok := ret.Get(0).(func(context.Context, int64, int, string) []byte); ok {
		r0 = rf(ctx, logIndex, usdcTokenIndexOffset, txHash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, int64, int, string) error); ok {
		r1 = rf(ctx, logIndex, usdcTokenIndexOffset, txHash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// USDCReader_GetUSDCMessagePriorToLogIndexInTx_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetUSDCMessagePriorToLogIndexInTx'
type USDCReader_GetUSDCMessagePriorToLogIndexInTx_Call struct {
	*mock.Call
}

// GetUSDCMessagePriorToLogIndexInTx is a helper method to define mock.On call
//   - ctx context.Context
//   - logIndex int64
//   - usdcTokenIndexOffset int
//   - txHash string
func (_e *USDCReader_Expecter) GetUSDCMessagePriorToLogIndexInTx(ctx interface{}, logIndex interface{}, usdcTokenIndexOffset interface{}, txHash interface{}) *USDCReader_GetUSDCMessagePriorToLogIndexInTx_Call {
	return &USDCReader_GetUSDCMessagePriorToLogIndexInTx_Call{Call: _e.mock.On("GetUSDCMessagePriorToLogIndexInTx", ctx, logIndex, usdcTokenIndexOffset, txHash)}
}

func (_c *USDCReader_GetUSDCMessagePriorToLogIndexInTx_Call) Run(run func(ctx context.Context, logIndex int64, usdcTokenIndexOffset int, txHash string)) *USDCReader_GetUSDCMessagePriorToLogIndexInTx_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(int64), args[2].(int), args[3].(string))
	})
	return _c
}

func (_c *USDCReader_GetUSDCMessagePriorToLogIndexInTx_Call) Return(_a0 []byte, _a1 error) *USDCReader_GetUSDCMessagePriorToLogIndexInTx_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *USDCReader_GetUSDCMessagePriorToLogIndexInTx_Call) RunAndReturn(run func(context.Context, int64, int, string) ([]byte, error)) *USDCReader_GetUSDCMessagePriorToLogIndexInTx_Call {
	_c.Call.Return(run)
	return _c
}

// NewUSDCReader creates a new instance of USDCReader. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewUSDCReader(t interface {
	mock.TestingT
	Cleanup(func())
}) *USDCReader {
	mock := &USDCReader{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}