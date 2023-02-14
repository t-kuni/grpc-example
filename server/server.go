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
	"sync"
)

type chatServer struct {
	chat.UnimplementedChatServer

	*joinedUsers
	*latestComments
	stateWatcher  *stateWatcherType
	colorSequence *colorSequenceType
}

type stateWatcherType struct {
	mu *sync.Mutex
	c  *sync.Cond
}

func newStateWatcher() *stateWatcherType {
	mu := new(sync.Mutex)
	return &stateWatcherType{
		mu: mu,
		c:  sync.NewCond(mu),
	}
}

func (s stateWatcherType) Wait() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.c.Wait()
}

func (s stateWatcherType) Broadcast() {
	s.c.Broadcast()
}

func (s chatServer) makeState() *chat.State {
	return &chat.State{
		JoinedUsers:    s.joinedUsers.joinedUsers,
		LatestComments: s.latestComments.comments,
	}
}

type colorSequenceType struct {
	next uint32
	mu   *sync.Mutex
}

func newColorSequenceType() *colorSequenceType {
	mu := new(sync.Mutex)
	return &colorSequenceType{
		mu:   mu,
		next: 0,
	}
}

func (c *colorSequenceType) Get() uint32 {
	c.mu.Lock()
	sec := c.next
	c.next++
	if c.next > 5 {
		c.next = 0
	}
	c.mu.Unlock()
	return sec
}

type joinedUsers struct {
	joinedUsers []*chat.User
	mu          sync.Mutex
}

func (u *joinedUsers) addUser(user *chat.User) {
	u.mu.Lock()
	u.joinedUsers = append(u.joinedUsers, user)
	u.mu.Unlock()
}

func (u *joinedUsers) deleteUser(user *chat.User) {
	u.mu.Lock()
	u.joinedUsers = lo.Filter[*chat.User](u.joinedUsers, func(item *chat.User, index int) bool {
		return item.Id != user.Id
	})
	u.mu.Unlock()
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
}

func (s chatServer) Join(ctx context.Context, profile *chat.Profile) (*chat.User, error) {
	user := &chat.User{
		Id:      uuid.New().String(),
		Profile: profile,
		Color:   s.colorSequence.Get(),
	}
	s.joinedUsers.addUser(user)

	systemComment := &chat.Comment{
		Body:            fmt.Sprintf("%s has entered the room.", user.Profile.Name),
		Commenter:       nil,
		IsSystemComment: true,
	}
	s.addComment(systemComment)

	s.stateWatcher.Broadcast()

	log.Printf("Join user. Name: %s", profile.Name)

	return user, nil
}

func (s chatServer) Leave(ctx context.Context, user *chat.User) (*empty.Empty, error) {
	s.joinedUsers.deleteUser(user)

	systemComment := &chat.Comment{
		Body:            fmt.Sprintf("%s has left the room.", user.Profile.Name),
		Commenter:       nil,
		IsSystemComment: true,
	}
	s.addComment(systemComment)

	s.stateWatcher.Broadcast()

	log.Printf("Leave user. Name: %s", user.Profile.Name)

	return &empty.Empty{}, nil
}

func (s chatServer) SendComment(ctx context.Context, comment *chat.Comment) (*empty.Empty, error) {
	comment.IsSystemComment = false
	s.latestComments.addComment(comment)

	log.Printf("Send comment. Name: %s, Comment: %s", comment.Commenter.Profile.Name, comment.Body)

	s.stateWatcher.Broadcast()

	return &empty.Empty{}, nil
}

func (s chatServer) WatchState(_ *empty.Empty, stream chat.Chat_WatchStateServer) error {
	for {
		err := stream.Send(s.makeState())
		if err != nil {
			return err
		}
		s.stateWatcher.Wait()
	}
}

func main() {
	port := 30000

	fmt.Print("\033[H\033[2J") // コンソールをクリア

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	server := chatServer{
		joinedUsers:    &joinedUsers{},
		latestComments: &latestComments{},
		stateWatcher:   newStateWatcher(),
		colorSequence:  newColorSequenceType(),
	}
	chat.RegisterChatServer(grpcServer, server)

	log.Printf("Start server listening on port %d.", port)

	err = grpcServer.Serve(lis)
	if err != nil {
		log.Fatal(err)
	}
}
