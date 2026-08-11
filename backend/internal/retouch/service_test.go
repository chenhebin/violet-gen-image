package retouch

import "testing"

func TestTicketNumber(t *testing.T) {
	value := ticketNumber("12345678-1234-1234-1234-123456789012")
	if len(value) != len("RT-20060102-12345678") {
		t.Fatalf("unexpected ticket number %q", value)
	}
}

func TestUnique(t *testing.T) {
	values := unique([]string{"a", "a", " ", "b"})
	if len(values) != 2 || values[0] != "a" || values[1] != "b" {
		t.Fatalf("unexpected values %#v", values)
	}
}
