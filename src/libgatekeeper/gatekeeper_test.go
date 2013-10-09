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

	v, err := c.Get("key.onetime")
	if err != nil {
		t.Fatal(err.Error())
	}
	if v != "onetimekey" {
		t.Fatal("key is: " + v)
	}

	err = c.AddAccess("key.onetime", "")
	if err != nil {
		t.Fatal(err.Error())
	}

	_, err = NewOneTimeClient("localhost:1337", "onetime")
	if err != nil {
		t.Fatal(err.Error())
	}
}
