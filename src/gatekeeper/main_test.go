package main

import (
    "errors"
    "testing"
)

func TestGetNewSet(t *testing.T) {
    g := NewGateKeeper()

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

func TestPermission(t *testing.T) {
    g := NewGateKeeper()

    err := g.New("name", "value", "key")
    if err != nil {
        t.Error(err)
    }

    err = g.AddAccess("name", "key", "keyother")
    if err != nil {
        t.Error(err)
    }

    v, err := g.Get("name", "keyother")
    if err != nil {
        t.Error(err)
    }
    if v  != "value" {
        t.Error(errors.New("key value is not value"))
    }

    err = g.RemoveAccess("name", "key", "keyother")
    if err != nil {
        t.Error(err)
    }

    v, err = g.Get("name", "keyother")
    if err == nil {
        t.Error(errors.New("No permission denied thrown"))
    }
}
