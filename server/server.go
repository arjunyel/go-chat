package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"sync"

	"fmt"

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

var directory struct { //directory of all groups with contained users
	sync.RWMutex //embeded global mutex
	groups       map[string][]userInfo
}

func doesGroupExist(group string) bool { //check if group already exists
	directory.RLock()
	defer directory.RUnlock()
	_, ok := directory.groups[group]
	return ok
}

func addUser(group string, user userInfo) { //add user to a group
	directory.Lock()
	defer directory.Unlock()
	dirct := directory.groups[group]
	dirct = append(dirct, user)
	directory.groups[group] = dirct
	return
}

func createGroup(group string, user userInfo) { //create a group and add the user
	directory.Lock()
	defer directory.Unlock()
	slice := []userInfo{user}
	directory.groups[group] = slice
	return
}
func register(group string, user userInfo) { //register client on first connect
	exist := doesGroupExist(group)
	if exist {
		addUser(group, user)
		return
	}
	createGroup(group, user)
	return
}

func sendMessage(message pb.ChatMessage) { //send message to group
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
func monitorOutbox(stream pb.GroupChat_ChatServer, message chan<- pb.ChatMessage) { //read in messages
	msg, err := stream.Recv()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(msg)
	message <- *msg
}

func (s *server) Chat(stream pb.GroupChat_ChatServer) error {
	in, err := stream.Recv() //listen to stream from client
	if err != nil {
		return err
	}
	inbox := make(chan pb.ChatMessage, 1000)
	if in.Message == "reg" { //register client on first connect
		register(in.Group, userInfo{in.Name, inbox})
	}
	outbox := make(chan pb.ChatMessage, 1000) //make channel for outgoing
	go monitorOutbox(stream, outbox)

	for { //route messages from channels
		select {
		case outgoing := <-outbox:
			fmt.Println("Sending message channel")
			sendMessage(outgoing)
		case incoming := <-inbox:
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
