package main

import "testing"

func Test_percentile(t *testing.T) {
	list := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	index := percentile(len(list), 0.5)

	if index != 5 {
		t.Fatalf("got=%d, want=5", index)
	}
}
