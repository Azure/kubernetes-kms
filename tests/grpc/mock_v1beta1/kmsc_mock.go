// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/Azure/kubernetes-kms/v1beta1 (interfaces: KeyManagementServiceClient)

// Package mock_v1beta1 is a generated GoMock package.
package mock_v1beta1

import (
	context "context"
	reflect "reflect"

	v1beta1 "github.com/Azure/kubernetes-kms/v1beta1"
	gomock "github.com/golang/mock/gomock"
	grpc "google.golang.org/grpc"
)

// MockKeyManagementServiceClient is a mock of KeyManagementServiceClient interface
type MockKeyManagementServiceClient struct {
	ctrl     *gomock.Controller
	recorder *MockKeyManagementServiceClientMockRecorder
}

// MockKeyManagementServiceClientMockRecorder is the mock recorder for MockKeyManagementServiceClient
type MockKeyManagementServiceClientMockRecorder struct {
	mock *MockKeyManagementServiceClient
}

// NewMockKeyManagementServiceClient creates a new mock instance
func NewMockKeyManagementServiceClient(ctrl *gomock.Controller) *MockKeyManagementServiceClient {
	mock := &MockKeyManagementServiceClient{ctrl: ctrl}
	mock.recorder = &MockKeyManagementServiceClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockKeyManagementServiceClient) EXPECT() *MockKeyManagementServiceClientMockRecorder {
	return m.recorder
}

// Decrypt mocks base method
func (m *MockKeyManagementServiceClient) Decrypt(arg0 context.Context, arg1 *v1beta1.DecryptRequest, arg2 ...grpc.CallOption) (*v1beta1.DecryptResponse, error) {
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Decrypt", varargs...)
	ret0, _ := ret[0].(*v1beta1.DecryptResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Decrypt indicates an expected call of Decrypt
func (mr *MockKeyManagementServiceClientMockRecorder) Decrypt(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Decrypt", reflect.TypeOf((*MockKeyManagementServiceClient)(nil).Decrypt), varargs...)
}

// Encrypt mocks base method
func (m *MockKeyManagementServiceClient) Encrypt(arg0 context.Context, arg1 *v1beta1.EncryptRequest, arg2 ...grpc.CallOption) (*v1beta1.EncryptResponse, error) {
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Encrypt", varargs...)
	ret0, _ := ret[0].(*v1beta1.EncryptResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Encrypt indicates an expected call of Encrypt
func (mr *MockKeyManagementServiceClientMockRecorder) Encrypt(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Encrypt", reflect.TypeOf((*MockKeyManagementServiceClient)(nil).Encrypt), varargs...)
}

// Version mocks base method
func (m *MockKeyManagementServiceClient) Version(arg0 context.Context, arg1 *v1beta1.VersionRequest, arg2 ...grpc.CallOption) (*v1beta1.VersionResponse, error) {
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Version", varargs...)
	ret0, _ := ret[0].(*v1beta1.VersionResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Version indicates an expected call of Version
func (mr *MockKeyManagementServiceClientMockRecorder) Version(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Version", reflect.TypeOf((*MockKeyManagementServiceClient)(nil).Version), varargs...)
}
