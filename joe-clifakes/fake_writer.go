// Code generated by counterfeiter. DO NOT EDIT.
package joeclifakes

import (
	"sync"

	cli "github.com/Carbonfrost/joe-cli"
)

type FakeWriter struct {
	ClearStyleStub        func(cli.Style)
	clearStyleMutex       sync.RWMutex
	clearStyleArgsForCall []struct {
		arg1 cli.Style
	}
	ResetStub        func()
	resetMutex       sync.RWMutex
	resetArgsForCall []struct {
	}
	ResetColorCapableStub        func()
	resetColorCapableMutex       sync.RWMutex
	resetColorCapableArgsForCall []struct {
	}
	SetBackgroundStub        func(cli.Color)
	setBackgroundMutex       sync.RWMutex
	setBackgroundArgsForCall []struct {
		arg1 cli.Color
	}
	SetColorCapableStub        func(bool)
	setColorCapableMutex       sync.RWMutex
	setColorCapableArgsForCall []struct {
		arg1 bool
	}
	SetForegroundStub        func(cli.Color)
	setForegroundMutex       sync.RWMutex
	setForegroundArgsForCall []struct {
		arg1 cli.Color
	}
	SetStyleStub        func(cli.Style)
	setStyleMutex       sync.RWMutex
	setStyleArgsForCall []struct {
		arg1 cli.Style
	}
	WriteStub        func([]byte) (int, error)
	writeMutex       sync.RWMutex
	writeArgsForCall []struct {
		arg1 []byte
	}
	writeReturns struct {
		result1 int
		result2 error
	}
	writeReturnsOnCall map[int]struct {
		result1 int
		result2 error
	}
	WriteStringStub        func(string) (int, error)
	writeStringMutex       sync.RWMutex
	writeStringArgsForCall []struct {
		arg1 string
	}
	writeStringReturns struct {
		result1 int
		result2 error
	}
	writeStringReturnsOnCall map[int]struct {
		result1 int
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeWriter) ClearStyle(arg1 cli.Style) {
	fake.clearStyleMutex.Lock()
	fake.clearStyleArgsForCall = append(fake.clearStyleArgsForCall, struct {
		arg1 cli.Style
	}{arg1})
	stub := fake.ClearStyleStub
	fake.recordInvocation("ClearStyle", []interface{}{arg1})
	fake.clearStyleMutex.Unlock()
	if stub != nil {
		fake.ClearStyleStub(arg1)
	}
}

func (fake *FakeWriter) ClearStyleCallCount() int {
	fake.clearStyleMutex.RLock()
	defer fake.clearStyleMutex.RUnlock()
	return len(fake.clearStyleArgsForCall)
}

func (fake *FakeWriter) ClearStyleCalls(stub func(cli.Style)) {
	fake.clearStyleMutex.Lock()
	defer fake.clearStyleMutex.Unlock()
	fake.ClearStyleStub = stub
}

func (fake *FakeWriter) ClearStyleArgsForCall(i int) cli.Style {
	fake.clearStyleMutex.RLock()
	defer fake.clearStyleMutex.RUnlock()
	argsForCall := fake.clearStyleArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeWriter) Reset() {
	fake.resetMutex.Lock()
	fake.resetArgsForCall = append(fake.resetArgsForCall, struct {
	}{})
	stub := fake.ResetStub
	fake.recordInvocation("Reset", []interface{}{})
	fake.resetMutex.Unlock()
	if stub != nil {
		fake.ResetStub()
	}
}

func (fake *FakeWriter) ResetCallCount() int {
	fake.resetMutex.RLock()
	defer fake.resetMutex.RUnlock()
	return len(fake.resetArgsForCall)
}

func (fake *FakeWriter) ResetCalls(stub func()) {
	fake.resetMutex.Lock()
	defer fake.resetMutex.Unlock()
	fake.ResetStub = stub
}

func (fake *FakeWriter) ResetColorCapable() {
	fake.resetColorCapableMutex.Lock()
	fake.resetColorCapableArgsForCall = append(fake.resetColorCapableArgsForCall, struct {
	}{})
	stub := fake.ResetColorCapableStub
	fake.recordInvocation("ResetColorCapable", []interface{}{})
	fake.resetColorCapableMutex.Unlock()
	if stub != nil {
		fake.ResetColorCapableStub()
	}
}

func (fake *FakeWriter) ResetColorCapableCallCount() int {
	fake.resetColorCapableMutex.RLock()
	defer fake.resetColorCapableMutex.RUnlock()
	return len(fake.resetColorCapableArgsForCall)
}

func (fake *FakeWriter) ResetColorCapableCalls(stub func()) {
	fake.resetColorCapableMutex.Lock()
	defer fake.resetColorCapableMutex.Unlock()
	fake.ResetColorCapableStub = stub
}

func (fake *FakeWriter) SetBackground(arg1 cli.Color) {
	fake.setBackgroundMutex.Lock()
	fake.setBackgroundArgsForCall = append(fake.setBackgroundArgsForCall, struct {
		arg1 cli.Color
	}{arg1})
	stub := fake.SetBackgroundStub
	fake.recordInvocation("SetBackground", []interface{}{arg1})
	fake.setBackgroundMutex.Unlock()
	if stub != nil {
		fake.SetBackgroundStub(arg1)
	}
}

func (fake *FakeWriter) SetBackgroundCallCount() int {
	fake.setBackgroundMutex.RLock()
	defer fake.setBackgroundMutex.RUnlock()
	return len(fake.setBackgroundArgsForCall)
}

func (fake *FakeWriter) SetBackgroundCalls(stub func(cli.Color)) {
	fake.setBackgroundMutex.Lock()
	defer fake.setBackgroundMutex.Unlock()
	fake.SetBackgroundStub = stub
}

func (fake *FakeWriter) SetBackgroundArgsForCall(i int) cli.Color {
	fake.setBackgroundMutex.RLock()
	defer fake.setBackgroundMutex.RUnlock()
	argsForCall := fake.setBackgroundArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeWriter) SetColorCapable(arg1 bool) {
	fake.setColorCapableMutex.Lock()
	fake.setColorCapableArgsForCall = append(fake.setColorCapableArgsForCall, struct {
		arg1 bool
	}{arg1})
	stub := fake.SetColorCapableStub
	fake.recordInvocation("SetColorCapable", []interface{}{arg1})
	fake.setColorCapableMutex.Unlock()
	if stub != nil {
		fake.SetColorCapableStub(arg1)
	}
}

func (fake *FakeWriter) SetColorCapableCallCount() int {
	fake.setColorCapableMutex.RLock()
	defer fake.setColorCapableMutex.RUnlock()
	return len(fake.setColorCapableArgsForCall)
}

func (fake *FakeWriter) SetColorCapableCalls(stub func(bool)) {
	fake.setColorCapableMutex.Lock()
	defer fake.setColorCapableMutex.Unlock()
	fake.SetColorCapableStub = stub
}

func (fake *FakeWriter) SetColorCapableArgsForCall(i int) bool {
	fake.setColorCapableMutex.RLock()
	defer fake.setColorCapableMutex.RUnlock()
	argsForCall := fake.setColorCapableArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeWriter) SetForeground(arg1 cli.Color) {
	fake.setForegroundMutex.Lock()
	fake.setForegroundArgsForCall = append(fake.setForegroundArgsForCall, struct {
		arg1 cli.Color
	}{arg1})
	stub := fake.SetForegroundStub
	fake.recordInvocation("SetForeground", []interface{}{arg1})
	fake.setForegroundMutex.Unlock()
	if stub != nil {
		fake.SetForegroundStub(arg1)
	}
}

func (fake *FakeWriter) SetForegroundCallCount() int {
	fake.setForegroundMutex.RLock()
	defer fake.setForegroundMutex.RUnlock()
	return len(fake.setForegroundArgsForCall)
}

func (fake *FakeWriter) SetForegroundCalls(stub func(cli.Color)) {
	fake.setForegroundMutex.Lock()
	defer fake.setForegroundMutex.Unlock()
	fake.SetForegroundStub = stub
}

func (fake *FakeWriter) SetForegroundArgsForCall(i int) cli.Color {
	fake.setForegroundMutex.RLock()
	defer fake.setForegroundMutex.RUnlock()
	argsForCall := fake.setForegroundArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeWriter) SetStyle(arg1 cli.Style) {
	fake.setStyleMutex.Lock()
	fake.setStyleArgsForCall = append(fake.setStyleArgsForCall, struct {
		arg1 cli.Style
	}{arg1})
	stub := fake.SetStyleStub
	fake.recordInvocation("SetStyle", []interface{}{arg1})
	fake.setStyleMutex.Unlock()
	if stub != nil {
		fake.SetStyleStub(arg1)
	}
}

func (fake *FakeWriter) SetStyleCallCount() int {
	fake.setStyleMutex.RLock()
	defer fake.setStyleMutex.RUnlock()
	return len(fake.setStyleArgsForCall)
}

func (fake *FakeWriter) SetStyleCalls(stub func(cli.Style)) {
	fake.setStyleMutex.Lock()
	defer fake.setStyleMutex.Unlock()
	fake.SetStyleStub = stub
}

func (fake *FakeWriter) SetStyleArgsForCall(i int) cli.Style {
	fake.setStyleMutex.RLock()
	defer fake.setStyleMutex.RUnlock()
	argsForCall := fake.setStyleArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeWriter) Write(arg1 []byte) (int, error) {
	var arg1Copy []byte
	if arg1 != nil {
		arg1Copy = make([]byte, len(arg1))
		copy(arg1Copy, arg1)
	}
	fake.writeMutex.Lock()
	ret, specificReturn := fake.writeReturnsOnCall[len(fake.writeArgsForCall)]
	fake.writeArgsForCall = append(fake.writeArgsForCall, struct {
		arg1 []byte
	}{arg1Copy})
	stub := fake.WriteStub
	fakeReturns := fake.writeReturns
	fake.recordInvocation("Write", []interface{}{arg1Copy})
	fake.writeMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeWriter) WriteCallCount() int {
	fake.writeMutex.RLock()
	defer fake.writeMutex.RUnlock()
	return len(fake.writeArgsForCall)
}

func (fake *FakeWriter) WriteCalls(stub func([]byte) (int, error)) {
	fake.writeMutex.Lock()
	defer fake.writeMutex.Unlock()
	fake.WriteStub = stub
}

func (fake *FakeWriter) WriteArgsForCall(i int) []byte {
	fake.writeMutex.RLock()
	defer fake.writeMutex.RUnlock()
	argsForCall := fake.writeArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeWriter) WriteReturns(result1 int, result2 error) {
	fake.writeMutex.Lock()
	defer fake.writeMutex.Unlock()
	fake.WriteStub = nil
	fake.writeReturns = struct {
		result1 int
		result2 error
	}{result1, result2}
}

func (fake *FakeWriter) WriteReturnsOnCall(i int, result1 int, result2 error) {
	fake.writeMutex.Lock()
	defer fake.writeMutex.Unlock()
	fake.WriteStub = nil
	if fake.writeReturnsOnCall == nil {
		fake.writeReturnsOnCall = make(map[int]struct {
			result1 int
			result2 error
		})
	}
	fake.writeReturnsOnCall[i] = struct {
		result1 int
		result2 error
	}{result1, result2}
}

func (fake *FakeWriter) WriteString(arg1 string) (int, error) {
	fake.writeStringMutex.Lock()
	ret, specificReturn := fake.writeStringReturnsOnCall[len(fake.writeStringArgsForCall)]
	fake.writeStringArgsForCall = append(fake.writeStringArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.WriteStringStub
	fakeReturns := fake.writeStringReturns
	fake.recordInvocation("WriteString", []interface{}{arg1})
	fake.writeStringMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeWriter) WriteStringCallCount() int {
	fake.writeStringMutex.RLock()
	defer fake.writeStringMutex.RUnlock()
	return len(fake.writeStringArgsForCall)
}

func (fake *FakeWriter) WriteStringCalls(stub func(string) (int, error)) {
	fake.writeStringMutex.Lock()
	defer fake.writeStringMutex.Unlock()
	fake.WriteStringStub = stub
}

func (fake *FakeWriter) WriteStringArgsForCall(i int) string {
	fake.writeStringMutex.RLock()
	defer fake.writeStringMutex.RUnlock()
	argsForCall := fake.writeStringArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeWriter) WriteStringReturns(result1 int, result2 error) {
	fake.writeStringMutex.Lock()
	defer fake.writeStringMutex.Unlock()
	fake.WriteStringStub = nil
	fake.writeStringReturns = struct {
		result1 int
		result2 error
	}{result1, result2}
}

func (fake *FakeWriter) WriteStringReturnsOnCall(i int, result1 int, result2 error) {
	fake.writeStringMutex.Lock()
	defer fake.writeStringMutex.Unlock()
	fake.WriteStringStub = nil
	if fake.writeStringReturnsOnCall == nil {
		fake.writeStringReturnsOnCall = make(map[int]struct {
			result1 int
			result2 error
		})
	}
	fake.writeStringReturnsOnCall[i] = struct {
		result1 int
		result2 error
	}{result1, result2}
}

func (fake *FakeWriter) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.clearStyleMutex.RLock()
	defer fake.clearStyleMutex.RUnlock()
	fake.resetMutex.RLock()
	defer fake.resetMutex.RUnlock()
	fake.resetColorCapableMutex.RLock()
	defer fake.resetColorCapableMutex.RUnlock()
	fake.setBackgroundMutex.RLock()
	defer fake.setBackgroundMutex.RUnlock()
	fake.setColorCapableMutex.RLock()
	defer fake.setColorCapableMutex.RUnlock()
	fake.setForegroundMutex.RLock()
	defer fake.setForegroundMutex.RUnlock()
	fake.setStyleMutex.RLock()
	defer fake.setStyleMutex.RUnlock()
	fake.writeMutex.RLock()
	defer fake.writeMutex.RUnlock()
	fake.writeStringMutex.RLock()
	defer fake.writeStringMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeWriter) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ cli.Writer = new(FakeWriter)
