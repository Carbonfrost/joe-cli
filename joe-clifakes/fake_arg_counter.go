// Code generated by counterfeiter. DO NOT EDIT.
package joeclifakes

import (
	"sync"

	cli "github.com/Carbonfrost/joe-cli"
)

type FakeArgCounter struct {
	DoneStub        func() error
	doneMutex       sync.RWMutex
	doneArgsForCall []struct {
	}
	doneReturns struct {
		result1 error
	}
	doneReturnsOnCall map[int]struct {
		result1 error
	}
	TakeStub        func(string, bool) error
	takeMutex       sync.RWMutex
	takeArgsForCall []struct {
		arg1 string
		arg2 bool
	}
	takeReturns struct {
		result1 error
	}
	takeReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeArgCounter) Done() error {
	fake.doneMutex.Lock()
	ret, specificReturn := fake.doneReturnsOnCall[len(fake.doneArgsForCall)]
	fake.doneArgsForCall = append(fake.doneArgsForCall, struct {
	}{})
	stub := fake.DoneStub
	fakeReturns := fake.doneReturns
	fake.recordInvocation("Done", []interface{}{})
	fake.doneMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeArgCounter) DoneCallCount() int {
	fake.doneMutex.RLock()
	defer fake.doneMutex.RUnlock()
	return len(fake.doneArgsForCall)
}

func (fake *FakeArgCounter) DoneCalls(stub func() error) {
	fake.doneMutex.Lock()
	defer fake.doneMutex.Unlock()
	fake.DoneStub = stub
}

func (fake *FakeArgCounter) DoneReturns(result1 error) {
	fake.doneMutex.Lock()
	defer fake.doneMutex.Unlock()
	fake.DoneStub = nil
	fake.doneReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeArgCounter) DoneReturnsOnCall(i int, result1 error) {
	fake.doneMutex.Lock()
	defer fake.doneMutex.Unlock()
	fake.DoneStub = nil
	if fake.doneReturnsOnCall == nil {
		fake.doneReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.doneReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeArgCounter) Take(arg1 string, arg2 bool) error {
	fake.takeMutex.Lock()
	ret, specificReturn := fake.takeReturnsOnCall[len(fake.takeArgsForCall)]
	fake.takeArgsForCall = append(fake.takeArgsForCall, struct {
		arg1 string
		arg2 bool
	}{arg1, arg2})
	stub := fake.TakeStub
	fakeReturns := fake.takeReturns
	fake.recordInvocation("Take", []interface{}{arg1, arg2})
	fake.takeMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeArgCounter) TakeCallCount() int {
	fake.takeMutex.RLock()
	defer fake.takeMutex.RUnlock()
	return len(fake.takeArgsForCall)
}

func (fake *FakeArgCounter) TakeCalls(stub func(string, bool) error) {
	fake.takeMutex.Lock()
	defer fake.takeMutex.Unlock()
	fake.TakeStub = stub
}

func (fake *FakeArgCounter) TakeArgsForCall(i int) (string, bool) {
	fake.takeMutex.RLock()
	defer fake.takeMutex.RUnlock()
	argsForCall := fake.takeArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeArgCounter) TakeReturns(result1 error) {
	fake.takeMutex.Lock()
	defer fake.takeMutex.Unlock()
	fake.TakeStub = nil
	fake.takeReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeArgCounter) TakeReturnsOnCall(i int, result1 error) {
	fake.takeMutex.Lock()
	defer fake.takeMutex.Unlock()
	fake.TakeStub = nil
	if fake.takeReturnsOnCall == nil {
		fake.takeReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.takeReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeArgCounter) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.doneMutex.RLock()
	defer fake.doneMutex.RUnlock()
	fake.takeMutex.RLock()
	defer fake.takeMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeArgCounter) recordInvocation(key string, args []interface{}) {
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

var _ cli.ArgCounter = new(FakeArgCounter)
