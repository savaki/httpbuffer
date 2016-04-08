# muxer

muxer illustrates how one can buffer requests to be sent to be backend,
presumably to improve overall throughput.

``` golang
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Given
	count := 10
	mux := httpbuffer.New(ctx, ProcessAll,
		httpbuffer.BatchSize(count),
		httpbuffer.Timeout(time.Second),
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
```