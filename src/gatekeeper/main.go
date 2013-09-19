package main

import (
    "errors"
	"io"
	"log"
	"net/http"
)


type gatekeeper struct {
    objects map[string] struct {
        value string
        owner string
        permissions map[string]bool
    }

    owners map[string]bool
}

func(g *gatekeeper) Get(item, key string) (value string, err error) {
    err = errors.New("No Such Item or Permission Denied")

    o, ok := g.objects[item]
    if !ok {
        return
    }

    ok = o.permissions[key]
    if !ok {
        return
    }

    return o.value, nil
}

func (g *gatekeeper) Set(item, key string) (err error) {
    err = errors.New("Permission Denied")

    ok := g.owners[key]
    if !ok {
        return
    }

    v, found := g.objects[item]

    if (found && v.owner != key) {
        return
    }

    v.value = item
    g.objects[item] = v
    return nil
}

func (g * gatekeeper) AddAccess(item, key, newkey string) (err error) {
    err = errors.New("Permission Denied")

    ok := g.owners[key]
    if !ok {
        return
    }

    v, found := g.objects[item]

    if (found && v.owner != key) {
        return
    }

    v.permissions[newkey] = true
    g.objects[item] = v
    return nil
}

func (g * gatekeeper) RemoveAccess(item, key, newkey string) (err error) {
    err = errors.New("Permission Denied")

    ok := g.owners[key]
    if !ok {
        return
    }

    v, found := g.objects[item]

    if (found && v.owner != key) {
        return
    }

    v.permissions[newkey] = false
    g.objects[item] = v
    return nil
}

func main() {

	http.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "gatekeeper v0")
	})

	log.Fatal(http.ListenAndServe(":800", nil))
}
