// Copyright (c) 2017 Nathan Ollerenshaw
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

// Package splash provides a Pool type for storing shared resource of some kind, such
// as a database handle.
package splash

import (
	"fmt"
	"time"
)

// Resource is a resource that you wish to store in a splash Pool.
type Resource interface{}

// Pool contains the structures representing the pool of things that
// you wish to share.
type Pool struct {
	resources         chan Resource
	minimum           int
	timeout           int
	allocator         func() (Resource, error)
	deallocator       func(Resource) error
	logErrorHandler   func(error)
	logMessageHandler func(string)
	quitCommand       chan bool
}

func (p *Pool) setMinimum(m int) error {
	p.minimum = m
	return nil
}

// Minimum sets the minimum number of
func Minimum(m int) func(p *Pool) error {
	return func(p *Pool) error {
		return p.setMinimum(m)
	}
}

// NewPool allocates a new Pool with a given capacity.
func NewPool(capacity int, options ...func(*Pool) error) (*Pool, error) {
	if capacity < 10 {
		return nil, fmt.Errorf("a pool must have a capacity of at least 10")
	}

	p := new(Pool)
	p.resources = make(chan Resource, capacity)
	p.minimum = capacity / 10
	p.logErrorHandler = func(e error) {
		fmt.Printf("splash ERROR: %v\n", e.Error())
	}
	p.logMessageHandler = func(m string) {
		fmt.Printf("splash INFO: %v\n", m)
	}
	p.quitCommand = make(chan bool)

	for _, option := range options {
		err := option(p)
		if err != nil {
			return nil, fmt.Errorf("error creating new pool: %v", err.Error())
		}
	}

	// Create initial set of resources
	for i := 0; i < p.minimum; i++ {
		r, err := p.allocator()
		if err == nil {
			p.logMessageHandler("resource allocated ")
		} else {
			p.logErrorHandler(fmt.Errorf("unable to initialize pool with allocator: %s", err.Error()))
		}
		p.resources <- r
	}

	// start the pool monitor goroutine. This routine is responsible for ensuring that
	go func() {
		for {
			select {
			case <-p.quitCommand:
				p.logMessageHandler("splash pool monitor exiting")
				return
			default:
				if len(p.resources) < p.minimum {
					r, err := p.allocator()
					if err == nil {
						p.logMessageHandler("resource allocated ")
					} else {
						p.logErrorHandler(fmt.Errorf("unable to initialize pool with allocator: %s", err.Error()))
					}
					p.resources <- r
				} else {
					time.Sleep(100 * time.Millisecond)
				}
			}
		}
	}()

	return p, nil
}

// Fetch will fetch an item from the pool. You are responsible to return it back to the pool
// when you are finished with Return(). If the pool is empty, a new item handle will be allocated
func (p *Pool) Fetch() (interface{}, error) {
	select {
	case i := <-p.resources:
		return i, nil
	default:
		// There are 0 items in the pool, so we will allocate one
		item, err := p.allocator()
		if err != nil {
			p.logErrorHandler(fmt.Errorf("unable to create resource with allocator: %s", err.Error()))
		}
		return item, nil
	}
}

// Return a given item to the pool.
func (p *Pool) Return(resource interface{}) {
	select {
	case p.resources <- resource:
		return
	default:
		// if we blocked returning the item to the channel, it's full. Just deallocate.
		// Do it in a goroutine so that we don't block that caller.

		go func() {
			err := p.deallocator(resource)
			if err != nil {
				p.logErrorHandler(fmt.Errorf("unable to return item to pool: %s", err.Error()))
			}
		}()
		return
	}
}

// GetAvailable will return the current number of items available in the pool.
func (p *Pool) GetAvailable() int {
	return len(p.resources)
}
