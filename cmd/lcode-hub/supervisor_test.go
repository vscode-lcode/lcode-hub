package main

import "testing"

func TestSupervisor(t *testing.T) {
	sup := NewSupervisor()
	h := sup.NewHandler()
	t.Log(h)
}
