package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/reactivex/rxgo/v2"
)

var streamCh chan rxgo.Item

func main() {
	go func() {
		router := mux.NewRouter()
		router.HandleFunc("/hello", hello).Methods("POST")
		err := http.ListenAndServe(":8000", router)

		if err != nil {
			return
		}
	}()

	ctx := context.Background()

	streamCh = make(chan rxgo.Item)

	timeoutCtx, cancelTimeout := context.WithTimeout(ctx, time.Hour*24)

	observable := rxgo.FromEventSource(streamCh, rxgo.WithBackPressureStrategy(rxgo.Drop))

	observable.DoOnCompleted(func() {
		fmt.Println("done")
	})
	observable.Distinct(func(ctx context.Context, i interface{}) (interface{}, error) {
		return fmt.Sprintf("%v", i), nil
	}).Debounce(rxgo.WithDuration(time.Millisecond * 1000)).DoOnNext(func(payload interface{}) {
		fmt.Println("payload: ", payload)
		if payload == "close" {
			close(streamCh)

		}
	})

	select {
	case <-timeoutCtx.Done():
		fmt.Println("timeout done")
		close(streamCh)
		//delete the observer node every 24h
	case <-ctx.Done():
		//parent context done
		fmt.Println("parent context done")
		cancelTimeout()
	}
	time.Sleep(time.Second * 60)

}
func IsClosed(ch chan rxgo.Item) bool {
	select {
	case <-ch:
		return true
	default:
	}

	return false
}
func hello(writer http.ResponseWriter, request *http.Request) {

	response := map[string]string{
		"Message":     "Success",
		"Description": "You've successfully written to the client",
	}

	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		panic(err)
	}
	if !IsClosed(streamCh) {
		streamCh <- rxgo.Of(string(body))
	} else {
		fmt.Println("Channel closed!")
	}

	err = json.NewEncoder(writer).Encode(response)

	if err != nil {
		log.Fatalln(err)
	}
}
