package main

import (
    "errors"
    "testing"
)

func TestGetNewSet(t *testing.T) {
    g := NewGateKeeper()
    g.owners["key"] = true

    err := g.New("name", "value", "key")
    if err != nil {
        t.Error(err)
    }

    err = g.Set("name", "value2", "key")
    if err != nil {
        t.Error(err)
    }

    v, err := g.Get("name", "key")
    if v  != "value2" {
        t.Error(errors.New("key value is not value2"))
    }
}
