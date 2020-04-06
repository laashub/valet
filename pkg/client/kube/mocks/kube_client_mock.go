// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/solo-io/valet/pkg/client/kube (interfaces: Client)

// Package mock_kube is a generated GoMock package.
package mock_kube

import (
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockClient is a mock of Client interface
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *MockClientMockRecorder
}

// MockClientMockRecorder is the mock recorder for MockClient
type MockClientMockRecorder struct {
	mock *MockClient
}

// NewMockClient creates a new mock instance
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &MockClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockClient) EXPECT() *MockClientMockRecorder {
	return m.recorder
}

// GetIngressAddress mocks base method
func (m *MockClient) GetIngressAddress(arg0, arg1, arg2 string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIngressAddress", arg0, arg1, arg2)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetIngressAddress indicates an expected call of GetIngressAddress
func (mr *MockClientMockRecorder) GetIngressAddress(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIngressAddress", reflect.TypeOf((*MockClient)(nil).GetIngressAddress), arg0, arg1, arg2)
}

// WaitUntilPodsRunning mocks base method
func (m *MockClient) WaitUntilPodsRunning(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WaitUntilPodsRunning", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// WaitUntilPodsRunning indicates an expected call of WaitUntilPodsRunning
func (mr *MockClientMockRecorder) WaitUntilPodsRunning(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitUntilPodsRunning", reflect.TypeOf((*MockClient)(nil).WaitUntilPodsRunning), arg0)
}