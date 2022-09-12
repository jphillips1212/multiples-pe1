package main

import (
	"fmt"
	"testing"
)

func TestSumMultiples(t *testing.T) {
	expected := 63
	testChan := make(chan int)

	go sumMultiples(3, 20, testChan)

	response := <-testChan

	fmt.Println(fmt.Sprintf("expected response is %d, actual response is %d", expected, response))
}
