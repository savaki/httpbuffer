package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/savaki/muxer"
	"golang.org/x/net/context"
)

func ProcessAll(items []interface{}) (interface{}, error) {
	time.Sleep(3 * time.Second)
	return "world", nil
}

func MakeRequest(mux *muxer.Muxer, wg *sync.WaitGroup, index int) {
	defer wg.Done()
	fmt.Printf("#%2d: Sent\n", index)
	resp, err := mux.Request("hello")
	fmt.Printf("#%2d: Received => %v %v\n", index, resp, err)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Given
	count := 10
	mux := muxer.New(ctx, ProcessAll,
		muxer.BatchSize(count),
		muxer.Timeout(time.Second),
	)
	go mux.Start()

	// When - Fan out requests
	wg := &sync.WaitGroup{}
	wg.Add(count)
	for i := 1; i <= count; i++ {
		go MakeRequest(mux, wg, i)
	}

	// Then - Wait for completion
	wg.Wait()
}
