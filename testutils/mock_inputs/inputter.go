// Code generated by MockGen. DO NOT EDIT.
// Source: ./internal/inputs/interfaces.go

// Package mock_inputs is a generated GoMock package.
package mock_inputs

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	clogger "github.com/sinkingpoint/clogger/internal/clogger"
)

// MockInputter is a mock of Inputter interface.
type MockInputter struct {
	ctrl     *gomock.Controller
	recorder *MockInputterMockRecorder
}

// MockInputterMockRecorder is the mock recorder for MockInputter.
type MockInputterMockRecorder struct {
	mock *MockInputter
}

// NewMockInputter creates a new mock instance.
func NewMockInputter(ctrl *gomock.Controller) *MockInputter {
	mock := &MockInputter{ctrl: ctrl}
	mock.recorder = &MockInputterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockInputter) EXPECT() *MockInputterMockRecorder {
	return m.recorder
}

// Kill mocks base method.
func (m *MockInputter) Kill() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Kill")
}

// Kill indicates an expected call of Kill.
func (mr *MockInputterMockRecorder) Kill() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Kill", reflect.TypeOf((*MockInputter)(nil).Kill))
}

// Run mocks base method.
func (m *MockInputter) Run(ctx context.Context, flushChan clogger.MessageChannel) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Run", ctx, flushChan)
	ret0, _ := ret[0].(error)
	return ret0
}

// Run indicates an expected call of Run.
func (mr *MockInputterMockRecorder) Run(ctx, flushChan interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockInputter)(nil).Run), ctx, flushChan)
}
