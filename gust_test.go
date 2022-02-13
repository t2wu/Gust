package gust

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type StateImpl struct { // interface State
	run           bool
	cargo         interface{}
	err           error
	cargoReceived interface{}
	nextState     State
	name          string
}

func (s *StateImpl) Exec(cargo interface{}) (State, interface{}, error) {
	s.cargoReceived = cargo
	s.run = true
	return s.nextState, s.cargo, s.err
}

func (s *StateImpl) Name() string {
	return s.name
}

type StateNoName struct { // interface State
	run           bool
	cargo         interface{}
	err           error
	cargoReceived interface{}
	nextState     State
	name          string
}

func (s *StateNoName) Exec(cargo interface{}) (State, interface{}, error) {
	s.cargoReceived = cargo
	s.run = true
	return s.nextState, s.cargo, s.err
}

func NewObserverImpl() *ObserverImpl {
	return &ObserverImpl{
		states: make([][]string, 0),
	}
}

type ObserverImpl struct {
	states [][]string
}

func (o *ObserverImpl) StateChanged(priorState string, nextState string) {
	o.states = append(o.states, []string{priorState, nextState})
}

func TestContain_SameHandler_Works(t *testing.T) {
	handlers := make([]State, 0)
	a := &StateImpl{}
	b := &StateImpl{}
	handlers = append(handlers, a)

	assert.True(t, contains(handlers, a))
	assert.False(t, contains(handlers, b))
}

func TestState_TraverselFromAToB_Works(t *testing.T) {
	// Construct A -> B
	b := &StateImpl{}
	a := &StateImpl{
		nextState: b,
		cargo:     2,
	}

	m := NewStateMachine()
	m.AddState(a)
	m.AddState(b)

	assert.Len(t, m.States, 2)

	err := m.Run(1, a)
	if !assert.Nil(t, err) {
		return
	}

	assert.True(t, a.run)
	assert.True(t, b.run)
	assert.Equal(t, 1, a.cargoReceived.(int))
	assert.Equal(t, 2, b.cargoReceived.(int))
}

func TestStateTraversel_StateHasError_HasError(t *testing.T) {
	// Construct A -> B
	b := &StateImpl{}
	a := &StateImpl{
		nextState: b,
		cargo:     2,
		err:       fmt.Errorf("some error"),
	}

	m := NewStateMachine()
	m.AddState(a)
	m.AddState(b)

	assert.Len(t, m.States, 2)

	err := m.Run(1, a)
	if !assert.Error(t, err) {
		return
	}
	assert.Equal(t, a.err, err)
}

func TestStateTraversel_FromAToBToDNotC_Works(t *testing.T) {
	// Construct
	//     B
	//   /   \
	// A      D, where D is the end state. Go from A, C, D and B not run
	//   \   /
	//     C
	d := &StateImpl{}
	c := &StateImpl{
		nextState: d,
		cargo:     4,
	}
	b := &StateImpl{
		nextState: d,
		cargo:     3,
	}
	a := &StateImpl{
		nextState: c,
		cargo:     2,
	}

	m := NewStateMachine()
	m.AddState(a)
	m.AddState(b)
	m.AddState(c)
	m.AddState(d)

	assert.Len(t, m.States, 4)

	err := m.Run(1, a)
	if !assert.Nil(t, err) {
		return
	}

	assert.True(t, a.run)
	assert.True(t, c.run)
	assert.True(t, d.run)
	assert.False(t, b.run)
	assert.Equal(t, 1, a.cargoReceived.(int))
	assert.Equal(t, 2, c.cargoReceived.(int))
	assert.Equal(t, 4, d.cargoReceived.(int))
}

func TestObserver_TransitionFromAToBToD_ReportStateTransitions(t *testing.T) {
	// Construct
	//     B
	//   /   \
	// A      D, where D is the end state. Go from A, C, D and B not run
	//   \   /
	//     C
	d := &StateImpl{
		name: "stateD",
	}
	c := &StateImpl{
		nextState: d,
		name:      "stateC",
	}
	b := &StateImpl{
		nextState: d,
		name:      "stateB",
	}
	a := &StateImpl{
		nextState: c,
		name:      "stateA",
	}

	o := NewObserverImpl()

	m := NewStateMachine()
	m.AddState(a)
	m.AddState(b)
	m.AddState(c)
	m.AddState(d)

	m.RegisterObservers(o)

	assert.Len(t, m.States, 4)

	err := m.Run(nil, a)
	if !assert.Nil(t, err) {
		return
	}

	if !assert.Len(t, o.states, 3) {
		return
	}

	assert.Equal(t, []string{"", "stateA"}, o.states[0])
	assert.Equal(t, []string{"stateA", "stateC"}, o.states[1])
	assert.Equal(t, []string{"stateC", "stateD"}, o.states[2])
}

func TestObserver_TransitionFromAToBToDToE_WithoutName_NotReportStateTransition(t *testing.T) {
	// Construct
	//     B
	//   /   \
	// A      D -- E, where E is the end state. B not run. C has no name. So report 0->A, D->E only
	//   \   /
	//     C
	e := &StateImpl{
		name: "stateE",
	}
	d := &StateImpl{
		nextState: e,
		name:      "stateD",
	}
	c := &StateNoName{
		nextState: d,
		name:      "stateC",
	}
	b := &StateImpl{
		nextState: d,
		name:      "stateB",
	}
	a := &StateImpl{
		nextState: c,
		name:      "stateA",
	}

	o := NewObserverImpl()

	m := NewStateMachine()
	m.AddState(a)
	m.AddState(b)
	m.AddState(c)
	m.AddState(d)
	m.AddState(e)
	m.RegisterObservers(o)

	err := m.Run(nil, a)
	if !assert.Nil(t, err) {
		return
	}

	if !assert.Len(t, o.states, 2) {
		return
	}

	assert.Equal(t, []string{"", "stateA"}, o.states[0])
	assert.Equal(t, []string{"stateD", "stateE"}, o.states[1]) // A->C not reported since C has no name
}

func TestObserver_TransitionFromAToBToD_AddTwoObserver_BothReceived(t *testing.T) {
	// Construct
	//     B
	//   /   \
	// A      D, where D is the end state. Go from A, C, D and B not run
	//   \   /
	//     C
	d := &StateImpl{
		name: "stateD",
	}
	c := &StateImpl{
		nextState: d,
		name:      "stateC",
	}
	b := &StateImpl{
		nextState: d,
		name:      "stateB",
	}
	a := &StateImpl{
		nextState: c,
		name:      "stateA",
	}

	o1 := NewObserverImpl()
	o2 := NewObserverImpl()

	m := NewStateMachine()
	m.AddState(a)
	m.AddState(b)
	m.AddState(c)
	m.AddState(d)

	m.RegisterObservers(o1)
	m.RegisterObservers(o2)

	err := m.Run(nil, a)
	if !assert.Nil(t, err) {
		return
	}

	if !assert.Len(t, o1.states, 3) {
		return
	}

	if !assert.Len(t, o2.states, 3) {
		return
	}

	assert.Equal(t, []string{"", "stateA"}, o1.states[0])
	assert.Equal(t, []string{"stateA", "stateC"}, o1.states[1])
	assert.Equal(t, []string{"stateC", "stateD"}, o1.states[2])

	assert.Equal(t, []string{"", "stateA"}, o2.states[0])
	assert.Equal(t, []string{"stateA", "stateC"}, o2.states[1])
	assert.Equal(t, []string{"stateC", "stateD"}, o2.states[2])
}

func TestObserver_TransitionFromAToBToD_RemoveOneObserver_OneReceived(t *testing.T) {
	// Construct
	//     B
	//   /   \
	// A      D, where D is the end state. Go from A, C, D and B not run
	//   \   /
	//     C
	d := &StateImpl{
		name: "stateD",
	}
	c := &StateImpl{
		nextState: d,
		name:      "stateC",
	}
	b := &StateImpl{
		nextState: d,
		name:      "stateB",
	}
	a := &StateImpl{
		nextState: c,
		name:      "stateA",
	}

	o1 := NewObserverImpl()
	o2 := NewObserverImpl()

	m := NewStateMachine()
	m.AddState(a)
	m.AddState(b)
	m.AddState(c)
	m.AddState(d)

	m.RegisterObservers(o1)
	m.RegisterObservers(o2)
	m.RemoveObserver(o1)

	err := m.Run(nil, a)
	if !assert.Nil(t, err) {
		return
	}

	if !assert.Len(t, o1.states, 0) {
		return
	}

	if !assert.Len(t, o2.states, 3) {
		return
	}

	assert.Equal(t, []string{"", "stateA"}, o2.states[0])
	assert.Equal(t, []string{"stateA", "stateC"}, o2.states[1])
	assert.Equal(t, []string{"stateC", "stateD"}, o2.states[2])
}

func TestObserver_GoToANextStateNotRegistered_ReturnsError(t *testing.T) {
	//     B
	//   /   \
	// A      D, where D is the end state. Go from A, C, D and B not run
	//   \   /
	//     C
	d := &StateImpl{}
	c := &StateImpl{
		nextState: d,
	}
	b := &StateImpl{
		nextState: d,
	}
	a := &StateImpl{
		nextState: c,
	}

	m := NewStateMachine()
	m.AddState(a)
	m.AddState(b)
	// m.AddState(c) // not registered
	m.AddState(d)

	err := m.Run(nil, a)
	assert.Error(t, err)
}
