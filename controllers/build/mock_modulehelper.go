// Code generated by MockGen. DO NOT EDIT.
// Source: modulehelper.go

// Package build is a generated GoMock package.
package build

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1alpha1 "github.com/qbarrand/oot-operator/api/v1alpha1"
)

// MockModuleHelper is a mock of ModuleHelper interface.
type MockModuleHelper struct {
	ctrl     *gomock.Controller
	recorder *MockModuleHelperMockRecorder
}

// MockModuleHelperMockRecorder is the mock recorder for MockModuleHelper.
type MockModuleHelperMockRecorder struct {
	mock *MockModuleHelper
}

// NewMockModuleHelper creates a new mock instance.
func NewMockModuleHelper(ctrl *gomock.Controller) *MockModuleHelper {
	mock := &MockModuleHelper{ctrl: ctrl}
	mock.recorder = &MockModuleHelperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockModuleHelper) EXPECT() *MockModuleHelperMockRecorder {
	return m.recorder
}

// ApplyBuildArgOverrides mocks base method.
func (m *MockModuleHelper) ApplyBuildArgOverrides(args []v1alpha1.BuildArg, overrides ...v1alpha1.BuildArg) []v1alpha1.BuildArg {
	m.ctrl.T.Helper()
	varargs := []interface{}{args}
	for _, a := range overrides {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ApplyBuildArgOverrides", varargs...)
	ret0, _ := ret[0].([]v1alpha1.BuildArg)
	return ret0
}

// ApplyBuildArgOverrides indicates an expected call of ApplyBuildArgOverrides.
func (mr *MockModuleHelperMockRecorder) ApplyBuildArgOverrides(args interface{}, overrides ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{args}, overrides...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ApplyBuildArgOverrides", reflect.TypeOf((*MockModuleHelper)(nil).ApplyBuildArgOverrides), varargs...)
}
