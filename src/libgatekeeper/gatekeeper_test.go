package libgatekeeper

import (
    "testing"
)

func TestGateKeeper(t *testing.T) {
    g := NewServer()
    err := g.Listen(":1337")

    c := NewClient("localhost:1337", "key")
    err = c.Set("key.onetime", "onetimekey")
    if err != nil {
        t.Fatal(err)
    }

}
