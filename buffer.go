package muxer

import (
	"sync"
	"time"

	"golang.org/x/net/context"
)

const (
	DefaultBatchSize = 100
	DefaultTimeout   = 5 * time.Second
)

type response struct {
	out interface{}
	err error
}

type request struct {
	in interface{}
	ch chan *response
}

type ProcessorFunc func([]interface{}) (interface{}, error)

type Muxer struct {
	ctx           context.Context
	cancel        func()
	inCh          chan *request
	timeout       time.Duration
	once          *sync.Once
	wg            *sync.WaitGroup
	batchSize     int
	processorFunc ProcessorFunc
}

func New(parent context.Context, processorFunc ProcessorFunc, configs ...func(*Muxer)) *Muxer {
	ctx, cancel := context.WithCancel(parent)
	m := &Muxer{
		ctx:           ctx,
		cancel:        cancel,
		inCh:          make(chan *request),
		once:          &sync.Once{},
		wg:            &sync.WaitGroup{},
		timeout:       DefaultTimeout,
		batchSize:     DefaultBatchSize,
		processorFunc: processorFunc,
	}

	for _, config := range configs {
		config(m)
	}

	return m
}

func Timeout(v time.Duration) func(*Muxer) {
	return func(m *Muxer) {
		m.timeout = v
	}
}

func BatchSize(v int) func(*Muxer) {
	return func(m *Muxer) {
		m.batchSize = v
	}
}

func (m *Muxer) Start() {
	m.once.Do(func() {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			m.worker()
		}()
	})

	m.wg.Wait()
}

func (m *Muxer) worker() {
	timeout := time.NewTimer(m.timeout)

	buffer := make([]*request, m.batchSize)
	offset := 0

	for {
		timeout.Reset(m.timeout)
		select {
		case <-m.ctx.Done():
			return

		case <-timeout.C:
			if offset > 0 {
				m.processAll(buffer[0:offset])
				offset = 0
			}

		case v := <-m.inCh:
			buffer[offset] = v
			offset++
			if offset == m.batchSize {
				m.processAll(buffer[0:offset])
				offset = 0
			}
		}
	}
}

func (m *Muxer) processAll(requests []*request) {
	count := len(requests)
	if count == 0 {
		return
	}

	args := make([]interface{}, count)
	for i := 0; i < count; i++ {
		args[i] = requests[i]
	}

	out, err := m.processorFunc(args)
	go func(requests []*request) {
		resp := &response{
			out: out,
			err: err,
		}

		for _, req := range requests {
			req.ch <- resp
		}
	}(requests)
}

func (m *Muxer) Request(in interface{}) (interface{}, error) {
	respCh := make(chan *response)
	defer close(respCh)

	m.inCh <- &request{
		in: in,
		ch: respCh,
	}

	resp := <-respCh
	return resp.out, resp.err
}
