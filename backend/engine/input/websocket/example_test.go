package websocket

import (
	"fmt"
)

func ExampleServer() {
	server := NewServer(":8080", nil)
	_ = server
	fmt.Println("ready")
	// Output: ready
}
