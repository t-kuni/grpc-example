package main

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/t-kuni/grpc-example/grpc/chat"
	"google.golang.org/grpc"
	"log"
	"net"
	"strings"
	"sync"
)

var (
	stateUpdated chan int
)

type chatServer struct {
	chat.UnimplementedChatServer

	*joinedUsers
	*latestComments
}

func (s chatServer) makeState() *chat.State {
	return &chat.State{
		JoinedUsers:    s.joinedUsers.joinedUsers,
		LatestComments: s.latestComments.comments,
	}
}

type joinedUsers struct {
	joinedUsers []*chat.User
	mu          sync.Mutex
}

func (u *joinedUsers) addUser(user *chat.User) {
	u.mu.Lock()
	u.joinedUsers = append(u.joinedUsers, user)
	u.mu.Unlock()

	stateUpdated <- 0
}

type latestComments struct {
	comments []*chat.Comment
	mu       sync.Mutex
}

func (u *latestComments) addComment(comment *chat.Comment) {
	u.mu.Lock()
	u.comments = append(u.comments, comment)

	if len(u.comments) > 10 {
		u.comments = u.comments[1:]
	}

	u.mu.Unlock()

	stateUpdated <- 0
}

func (s chatServer) Join(ctx context.Context, profile *chat.Profile) (*chat.User, error) {
	user := &chat.User{
		Id:      uuid.New().String(),
		Profile: profile,
	}
	s.joinedUsers.addUser(user)

	s.render()

	return user, nil
}

func (s chatServer) SendComment(ctx context.Context, comment *chat.Comment) (*empty.Empty, error) {
	s.latestComments.addComment(comment)

	s.render()

	return &empty.Empty{}, nil
}

func (s chatServer) WatchState(_ *empty.Empty, stream chat.Chat_WatchStateServer) error {
	for {
		err := stream.Send(s.makeState())
		if err != nil {
			return err
		}
		select {
		case <-stateUpdated:
			log.Print("stateUpdated")
		}
	}
}

func (s chatServer) render() error {
	//fmt.Print("\033[u\033[K") // restore the cursor position and clear the line
	fmt.Print("\033[H\033[2J")

	fmt.Println("State: Running")

	userNameList := lo.Map[*chat.User, string](s.joinedUsers.joinedUsers, func(user *chat.User, index int) string {
		return user.Profile.Name
	})

	userNameText := strings.Join(userNameList, ", ")

	fmt.Println("Joined Users: " + userNameText)

	commentBodyList := lo.Map[*chat.Comment, string](s.latestComments.comments, func(comment *chat.Comment, index int) string {
		return fmt.Sprintf("[%s] %s", comment.Commenter.Profile.Name, comment.Body)
	})

	commentText := strings.Join(commentBodyList, "\n")

	fmt.Println(commentText)

	return nil
}

func (s chatServer) prepareRendering() {
	fmt.Print("\033[s") // save the cursor position
	fmt.Print("\033[H\033[2J")
}

func main() {
	stateUpdated = make(chan int, 10)

	port := 30000

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	server := chatServer{
		joinedUsers:    &joinedUsers{},
		latestComments: &latestComments{},
	}
	chat.RegisterChatServer(grpcServer, server)

	server.render()

	err = grpcServer.Serve(lis)
	if err != nil {
		log.Fatal(err)
	}
}
