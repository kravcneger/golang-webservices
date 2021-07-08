package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func ExecutePipeline(jobs []job) {
	var channels []chan interface{}
	for i := 0; i < len(jobs)+1; i++ {
		channels = append(channels, make(chan interface{}))
	}

	var wg = &sync.WaitGroup{}
	for i := 0; i < len(jobs); i++ {
		wg.Add(1)
		go func(i int, w *sync.WaitGroup, in, out chan interface{}) {
			defer w.Done()
			defer close(out)
			jobs[i](in, out)
		}(i, wg, channels[i], channels[i+1])
	}

	wg.Wait()
	close(channels[0])
}

func SingleHash(in_main, out_main chan interface{}) {
	var data []string
	for val := range in_main {
		data = append(data, fmt.Sprintf("%v", val))
	}
	first_out_channel := make(chan interface{})
	second_out_channel := make(chan interface{})

	freeFlowJobs1 := []job{
		job(func(in, out chan interface{}) {
			for _, val := range data {
				out <- DataSignerMd5(fmt.Sprintf("%v", val))
			}
		}),
		job(func(in, out chan interface{}) {
			var wg = &sync.WaitGroup{}
			var resChQueue [](chan interface{})
			num := 0
			for val := range in {
				resChQueue = append(resChQueue, make(chan interface{}, 1))
				wg.Add(1)
				go func(o chan interface{}, v interface{}, w *sync.WaitGroup) {
					defer w.Done()
					o <- DataSignerCrc32(fmt.Sprintf("%v", v))
				}(resChQueue[num], val, wg)
				num++
			}
			wg.Wait()
			for _, v := range resChQueue {
				first_out_channel <- fmt.Sprintf("%v", <-v)
				close(v)
			}
		}),
	}
	freeFlowJobs2 := []job{job(func(in, out chan interface{}) {
		var wg = &sync.WaitGroup{}
		var resChQueue []chan interface{}
		num := 0
		for _, val := range data {
			resChQueue = append(resChQueue, make(chan interface{}, 1))
			wg.Add(1)
			go func(o chan interface{}, v interface{}, default_data string, w *sync.WaitGroup) {
				defer w.Done()
				o <- DataSignerCrc32(default_data)
			}(resChQueue[num], val, data[num], wg)
			num++
		}
		wg.Wait()

		for _, v := range resChQueue {
			second_out_channel <- fmt.Sprintf("%v", <-v)
			close(v)
		}
	})}

	go ExecutePipeline(freeFlowJobs1)
	go ExecutePipeline(freeFlowJobs2)

	for _ = range data {
		out_main <- fmt.Sprintf("%v", <-second_out_channel) + "~" + fmt.Sprintf("%v", <-first_out_channel)
	}
}

func MultiHash(in_main, out_main chan interface{}) {
	freeFlowJobs := []job{
		job(func(in, out chan interface{}) {
			var in_data []interface{}

			for v := range in_main {
				in_data = append(in_data, v)
			}
			var wgExt = &sync.WaitGroup{}
			var resExtChQueue []chan interface{}
			num := 0
			for _, val := range in_data {
				resExtChQueue = append(resExtChQueue, make(chan interface{}, 1))
				wgExt.Add(1)

				go func(oEx chan interface{}, vExt interface{}, w *sync.WaitGroup) {
					defer wgExt.Done()
					data := ""
					var wg = &sync.WaitGroup{}
					var resChQueue []chan interface{}

					for i := 0; i <= 5; i++ {
						resChQueue = append(resChQueue, make(chan interface{}, 1))
						wg.Add(1)
						go func(o chan interface{}, v interface{}, w *sync.WaitGroup, n int) {
							defer w.Done()
							o <- DataSignerCrc32(strconv.Itoa(n) + fmt.Sprintf("%v", v))
						}(resChQueue[i], vExt, wg, i)
					}
					wg.Wait()
					for _, v := range resChQueue {
						data += fmt.Sprintf("%v", <-v)
					}
					oEx <- data
				}(resExtChQueue[num], val, wgExt)
				num++
			}
			wgExt.Wait()
			for _, v := range resExtChQueue {
				out_main <- fmt.Sprintf("%v", <-v)
				close(v)
			}
		}),
	}
	ExecutePipeline(freeFlowJobs)
}

func CombineResults(in_main, out_main chan interface{}) {
	var data []string
	for val := range in_main {
		data = append(data, fmt.Sprintf("%v", val))
	}
	sort.Slice(data, func(i, j int) bool {
		return data[i] < data[j]
	})
	res := strings.Join(data[:], "_")
	out_main <- res
}
