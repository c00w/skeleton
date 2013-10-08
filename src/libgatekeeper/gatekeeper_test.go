package libgatekeeper

import (
    "testing"
)

func TestGateKeeper(t *testing.T) {
    g := NewServer()
    go func() {
        err := g.Listen(":1337")
        t.Fatal(err)
    }()

    c := NewClient("localhost:1337", "key")
    err := c.New("key.onetime", "onetimekey")
    if err != nil {
        t.Fatal(err)
    }
}
