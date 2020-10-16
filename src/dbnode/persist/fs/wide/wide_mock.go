// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/m3db/m3/src/dbnode/persist/fs/wide (interfaces: IndexChecksumBlockBatchReader,StreamedMismatchBatch)

// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Package wide is a generated GoMock package.
package wide

import (
	"reflect"

	"github.com/golang/mock/gomock"
)

// MockIndexChecksumBlockBatchReader is a mock of IndexChecksumBlockBatchReader interface
type MockIndexChecksumBlockBatchReader struct {
	ctrl     *gomock.Controller
	recorder *MockIndexChecksumBlockBatchReaderMockRecorder
}

// MockIndexChecksumBlockBatchReaderMockRecorder is the mock recorder for MockIndexChecksumBlockBatchReader
type MockIndexChecksumBlockBatchReaderMockRecorder struct {
	mock *MockIndexChecksumBlockBatchReader
}

// NewMockIndexChecksumBlockBatchReader creates a new mock instance
func NewMockIndexChecksumBlockBatchReader(ctrl *gomock.Controller) *MockIndexChecksumBlockBatchReader {
	mock := &MockIndexChecksumBlockBatchReader{ctrl: ctrl}
	mock.recorder = &MockIndexChecksumBlockBatchReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockIndexChecksumBlockBatchReader) EXPECT() *MockIndexChecksumBlockBatchReaderMockRecorder {
	return m.recorder
}

// Current mocks base method
func (m *MockIndexChecksumBlockBatchReader) Current() IndexChecksumBlockBatch {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Current")
	ret0, _ := ret[0].(IndexChecksumBlockBatch)
	return ret0
}

// Current indicates an expected call of Current
func (mr *MockIndexChecksumBlockBatchReaderMockRecorder) Current() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Current", reflect.TypeOf((*MockIndexChecksumBlockBatchReader)(nil).Current))
}

// Next mocks base method
func (m *MockIndexChecksumBlockBatchReader) Next() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Next")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Next indicates an expected call of Next
func (mr *MockIndexChecksumBlockBatchReaderMockRecorder) Next() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Next", reflect.TypeOf((*MockIndexChecksumBlockBatchReader)(nil).Next))
}

// MockStreamedMismatchBatch is a mock of StreamedMismatchBatch interface
type MockStreamedMismatchBatch struct {
	ctrl     *gomock.Controller
	recorder *MockStreamedMismatchBatchMockRecorder
}

// MockStreamedMismatchBatchMockRecorder is the mock recorder for MockStreamedMismatchBatch
type MockStreamedMismatchBatchMockRecorder struct {
	mock *MockStreamedMismatchBatch
}

// NewMockStreamedMismatchBatch creates a new mock instance
func NewMockStreamedMismatchBatch(ctrl *gomock.Controller) *MockStreamedMismatchBatch {
	mock := &MockStreamedMismatchBatch{ctrl: ctrl}
	mock.recorder = &MockStreamedMismatchBatchMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockStreamedMismatchBatch) EXPECT() *MockStreamedMismatchBatchMockRecorder {
	return m.recorder
}

// RetrieveMismatchBatch mocks base method
func (m *MockStreamedMismatchBatch) RetrieveMismatchBatch() (ReadMismatchBatch, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RetrieveMismatchBatch")
	ret0, _ := ret[0].(ReadMismatchBatch)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RetrieveMismatchBatch indicates an expected call of RetrieveMismatchBatch
func (mr *MockStreamedMismatchBatchMockRecorder) RetrieveMismatchBatch() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RetrieveMismatchBatch", reflect.TypeOf((*MockStreamedMismatchBatch)(nil).RetrieveMismatchBatch))
}
