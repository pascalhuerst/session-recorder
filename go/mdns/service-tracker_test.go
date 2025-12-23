package mdns

import (
	"net"
	"testing"
)

func TestServiceAddress_String_IPv4(t *testing.T) {
	sa := ServiceAddress{
		IP:          net.ParseIP("192.168.1.100"),
		Port:        8080,
		ServiceName: "test-service",
	}

	got, err := sa.String()
	if err != nil {
		t.Errorf("ServiceAddress.String() error = %v", err)
		return
	}

	want := "192.168.1.100:8080"
	if got != want {
		t.Errorf("ServiceAddress.String() = %v, want %v", got, want)
	}
}

func TestServiceAddress_String_IPv6(t *testing.T) {
	sa := ServiceAddress{
		IP:          net.ParseIP("2001:db8::1"),
		Port:        8080,
		ServiceName: "test-service",
	}

	got, err := sa.String()
	if err != nil {
		t.Errorf("ServiceAddress.String() error = %v", err)
		return
	}

	want := "[2001:db8::1]:8080"
	if got != want {
		t.Errorf("ServiceAddress.String() = %v, want %v", got, want)
	}
}

func TestServiceAddress_String_IPv4Mapped(t *testing.T) {
	// IPv4-mapped IPv6 address should be treated as IPv4
	sa := ServiceAddress{
		IP:          net.ParseIP("::ffff:192.168.1.1"),
		Port:        3000,
		ServiceName: "test-service",
	}

	got, err := sa.String()
	if err != nil {
		t.Errorf("ServiceAddress.String() error = %v", err)
		return
	}

	// Should format as IPv4
	want := "192.168.1.1:3000"
	if got != want {
		t.Errorf("ServiceAddress.String() = %v, want %v", got, want)
	}
}

func TestServiceAddress_equal(t *testing.T) {
	tests := []struct {
		name  string
		sa1   ServiceAddress
		sa2   ServiceAddress
		equal bool
	}{
		{
			name: "identical addresses",
			sa1: ServiceAddress{
				IP:             net.ParseIP("192.168.1.1"),
				Port:           8080,
				InterfaceIndex: 1,
				ServiceName:    "test",
			},
			sa2: ServiceAddress{
				IP:             net.ParseIP("192.168.1.1"),
				Port:           8080,
				InterfaceIndex: 1,
				ServiceName:    "test",
			},
			equal: true,
		},
		{
			name: "different IP",
			sa1: ServiceAddress{
				IP:             net.ParseIP("192.168.1.1"),
				Port:           8080,
				InterfaceIndex: 1,
				ServiceName:    "test",
			},
			sa2: ServiceAddress{
				IP:             net.ParseIP("192.168.1.2"),
				Port:           8080,
				InterfaceIndex: 1,
				ServiceName:    "test",
			},
			equal: false,
		},
		{
			name: "different port",
			sa1: ServiceAddress{
				IP:             net.ParseIP("192.168.1.1"),
				Port:           8080,
				InterfaceIndex: 1,
				ServiceName:    "test",
			},
			sa2: ServiceAddress{
				IP:             net.ParseIP("192.168.1.1"),
				Port:           9090,
				InterfaceIndex: 1,
				ServiceName:    "test",
			},
			equal: false,
		},
		{
			name: "different service name",
			sa1: ServiceAddress{
				IP:             net.ParseIP("192.168.1.1"),
				Port:           8080,
				InterfaceIndex: 1,
				ServiceName:    "test1",
			},
			sa2: ServiceAddress{
				IP:             net.ParseIP("192.168.1.1"),
				Port:           8080,
				InterfaceIndex: 1,
				ServiceName:    "test2",
			},
			equal: false,
		},
		{
			name: "different interface",
			sa1: ServiceAddress{
				IP:             net.ParseIP("192.168.1.1"),
				Port:           8080,
				InterfaceIndex: 1,
				ServiceName:    "test",
			},
			sa2: ServiceAddress{
				IP:             net.ParseIP("192.168.1.1"),
				Port:           8080,
				InterfaceIndex: 2,
				ServiceName:    "test",
			},
			equal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sa1.equal(tt.sa2); got != tt.equal {
				t.Errorf("ServiceAddress.equal() = %v, want %v", got, tt.equal)
			}
		})
	}
}

func TestKeyForService(t *testing.T) {
	// This test requires importing avahi which has external dependencies
	// Skip if avahi is not available or test the function signature
	t.Skip("keyForService requires avahi.Service type")
}
