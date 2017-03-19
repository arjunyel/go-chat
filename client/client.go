package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/net/context"

	pb "github.com/arjunyel/go-chat"
	"google.golang.org/grpc"
)

const (
	port = ":12893"
)

func listen(stream pb.GroupChat_ChatClient, inbox chan pb.ChatMessage) {
	msg, _ := stream.Recv()
	inbox <- *msg
}

func send(outbox chan pb.ChatMessage, r *bufio.Reader, name string, group string) {
	msg, _ := r.ReadString('\n')
	msg = strings.TrimSpace(msg)
	outbox <- pb.ChatMessage{Name: name, Message: msg, Group: group}
}

func main() {
	r := bufio.NewReader(os.Stdin)

	// Read the server address
	fmt.Print("Please specify the server IP: ")
	address, _ := r.ReadString('\n')
	address = strings.TrimSpace(address)
	address = address + port

	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())

	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	}

	// Close the connection after main returns.
	defer conn.Close()

	// Create the client
	c := pb.NewGroupChatClient(conn)

	fmt.Printf("\nYou have successfully connected to %s! To disconnect, hit ctrl+c or type exit.\n", address)
	fmt.Println("Enter your name: ")
	name, err := r.ReadString('\n')
	if err != nil {
		return
	}
	name = strings.TrimSpace(name)
	fmt.Println("\nEnter your group: ")
	group, err := r.ReadString('\n')
	if err != nil {
		return
	}
	group = strings.TrimSpace(group)
	stream, err := c.Chat(context.Background())
	if err != nil {
		return
	}

	//Register client on server
	stream.Send(&pb.ChatMessage{Name: name, Message: "reg", Group: group})
	// Keep connection alive until ctrl+c or exit is entered.

	inbox := make(chan pb.ChatMessage, 1000)
	go listen(stream, inbox)
	outbox := make(chan pb.ChatMessage, 1000)
	go send(outbox, r, name, group)

	for {
		select {
		case sending := <-outbox:
			stream.Send(&sending)
		case receive := <-inbox:
			fmt.Printf("%s - %s\n", receive.Name, receive.Message)
		}
	}
}
