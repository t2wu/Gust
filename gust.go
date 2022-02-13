package gust

import (
	"fmt"
	"sync"
)

// Inspired by David Mertz's state machine in Python

// State is an interface which specifies a state with handler
type State interface {
	// Exec is the function to be executed when transitioned to the said state
	// cargo is anything that's passed from the previous state (or given in Run()
	// at the starting state). The function returns nextState object, the next cargo
	// for the next state, and any error if encountered
	Exec(cargo interface{}) (nextState State, nextCargo interface{}, err error)
}

// HaveName when implemented allows state to be reported during transition change
type HaveName interface {
	Name() string // state name, used in state change notification if needed
}

// Observer interface for observing any state change, if needed
type Observer interface {
	// StateChanged notifies the prior and the next string name, if the next
	// state is the starting state, the priorState is given an empy string
	StateChanged(priorState string, nextState string)
}

// NewStateMachine is a constructor for StateMachine
func NewStateMachine() *StateMachine {
	return &StateMachine{
		States:        make([]State, 0),
		observers:     make([]Observer, 0),
		observersLock: &sync.RWMutex{},
	}
}

// StateMachine is a handler for joggling between the states
type StateMachine struct {
	States []State

	observers     []Observer
	observersLock *sync.RWMutex
}

// RegisterObserver for any notification of state change event in between state change. When a state
// is executed, an event is sent
func (sm *StateMachine) RegisterObservers(os ...Observer) {
	sm.observersLock.Lock()
	defer sm.observersLock.Unlock()

	sm.observers = append(sm.observers, os...)
}

// RemoveObserver removes the observer from the observer list
func (sm *StateMachine) RemoveObserver(o Observer) {
	sm.observersLock.Lock()
	defer sm.observersLock.Unlock()

	indexToRemove := -1
	for i, observer := range sm.observers {
		if observer == o {
			indexToRemove = i
		}
	}
	if indexToRemove != -1 {
		sm.observers[indexToRemove] = sm.observers[len(sm.observers)-1] // move the last one over
		sm.observers[len(sm.observers)-1] = nil
		sm.observers = sm.observers[:len(sm.observers)-1] // truncate
	}
}

// AddState adds a state state
func (sm *StateMachine) AddState(state State) {
	sm.States = append(sm.States, state)
}

// Run starts the state machine from the start state
func (sm *StateMachine) Run(cargo interface{}, startState State) error {
	state := startState
	var priorState State = nil

	for {
		sm.NotifyState(priorState, state)
		nextState, nextCargo, err := state.Exec(cargo)
		if err != nil {
			return err
		}
		if nextState == nil {
			break
		}

		if !contains(sm.States, nextState) {
			return fmt.Errorf("invalid target state %v", nextState)
		} else {
			cargo = nextCargo
			priorState = state
			state = nextState
		}
	}

	return nil
}

// NotifyState notifies the observer about the state change
func (sm *StateMachine) NotifyState(prior, next State) {
	sm.observersLock.Lock()
	defer sm.observersLock.Unlock()

	for _, observer := range sm.observers {
		priorName, nextName := "", ""
		if n, ok := next.(HaveName); ok {
			nextName = n.Name()
		}
		if prior != nil {
			if p, ok := prior.(HaveName); ok {
				priorName = p.Name()
			} else {
				return
			}
		}

		if nextName != "" {
			observer.StateChanged(priorName, nextName)
		}
	}
}

func contains(s []State, e State) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
