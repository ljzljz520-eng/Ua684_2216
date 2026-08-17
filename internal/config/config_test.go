package config

import "testing"

func TestConfigDefaultsValidate(t *testing.T) {
	c := Default()
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if c.QueueSize != 32 {
		t.Fatalf("queue=%d", c.QueueSize)
	}
}
