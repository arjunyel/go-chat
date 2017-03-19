package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"sync"

	"fmt"

	"strconv"

	pb "github.com/arjunyel/go-chat"
)

const (
	port = ":12893"
)

type server struct{}

type userInfo struct {
	name    string
	channel chan pb.ChatMessage
}

var directory struct {
	sync.RWMutex
	groups map[string][]userInfo
}

func doesGroupExist(group string) bool {
	directory.RLock()
	defer directory.RUnlock()
	_, ok := directory.groups[group]
	return ok
}

func addUser(group string, user userInfo) {
	directory.Lock()
	defer directory.Unlock()
	dirct := directory.groups[group]
	dirct = append(dirct, user)
	directory.groups[group] = dirct
	fmt.Println(directory.groups)
}

func createGroup(group string, user userInfo) {
	directory.Lock()
	defer directory.Unlock()
	slice := []userInfo{user}
	directory.groups[group] = slice
	return
}
func register(group string, user userInfo) {
	exist := doesGroupExist(group)
	if exist {
		addUser(group, user)
		return
	}
	createGroup(group, user)
	fmt.Println(directory.groups)
	return

}

func sendMessage(message pb.ChatMessage) {
	directory.Lock()
	defer directory.Unlock()
	fmt.Println("sending " + message.Message + " from " + message.Name + " to " + message.Group)
	userList := directory.groups[message.Group]
	for _, user := range userList {
		if user.name != message.Name {
			user.channel <- message
		}
	}

}
func monitorOutbox(stream pb.GroupChat_ChatServer, message chan<- pb.ChatMessage) {
	msg, err := stream.Recv()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(msg)
	message <- *msg
}

func (s *server) Chat(stream pb.GroupChat_ChatServer) error {
	in, err := stream.Recv()
	if err != nil {
		return err
	}
	inbox := make(chan pb.ChatMessage, 1000)
	if in.Message == "reg" { /*Register the client*/
		register(in.Group, userInfo{in.Name, inbox})
	}
	outbox := make(chan pb.ChatMessage, 1000)
	go monitorOutbox(stream, outbox)
	var u = directory.groups[in.Group]
	var index int
	for i, user := range u {
		if in.Name == user.name {
			fmt.Println("found " + user.name + " at index " + strconv.Itoa(i))
			index = i
		}
	}
	for {
		select {
		case outgoing := <-outbox:
			fmt.Println("Sending message channel")
			sendMessage(outgoing)
		case incoming := <-directory.groups[in.Group][index].channel:
			fmt.Println(directory.groups[in.Group][index].channel)
			stream.Send(&incoming)
		}
	}
}

func main() {
	lis, err := net.Listen("tcp", port)

	if err != nil {
		log.Fatalf("Failed to listen %v", err)
	}

	directory.groups = make(map[string][]userInfo)

	// Initializes the gRPC server.
	s := grpc.NewServer()

	// Register the server with gRPC.
	pb.RegisterGroupChatServer(s, &server{})

	// Register reflection service on gRPC server.
	reflection.Register(s)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
