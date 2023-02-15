package domain

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/t-kuni/grpc-example/client/presenter"
	"github.com/t-kuni/grpc-example/grpc/chat"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
)

type App struct {
	client  chat.ChatClient
	conn    *grpc.ClientConn
	context context.Context
	user    *chat.User
	state   *chat.State
	ui      *presenter.UI
}

func NewApp() *App {
	return &App{
		context: context.Background(),
	}
}

func (a *App) ConnectUI(ui *presenter.UI) {
	ui.ViewModel.QuitHandler = a.OnQuit
	ui.ViewModel.SendCommentHandler = a.OnSendComment

	a.ui = ui
}

func (a *App) ConnectServer(host string, port int) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	a.conn = conn
	a.client = chat.NewChatClient(conn)
}

func (a *App) Join(name string) {
	profile := &chat.Profile{
		Name:   name,
		Age:    10,
		Gender: chat.Gender_GENDER_MAN,
	}
	user, err := a.client.Join(a.context, profile)
	if err != nil {
		log.Fatal(err)
	}
	a.user = user
}

func (a *App) StartWatchState() {
	go a.watchState()
}

func (a *App) watchState() {
	stream, err := a.client.WatchState(a.context, &empty.Empty{})
	if err != nil {
		log.Fatal(err)
	}
	for {
		state, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		a.state = state
		a.ui.Send(presenter.StateUpdatedMsg{
			state,
		})
	}
}

func (a *App) OnQuit() {
	if a.user != nil {
		a.client.Leave(a.context, a.user)
	}
}

func (a *App) OnSendComment(commentBody string) {
	comment := &chat.Comment{
		Body:      commentBody,
		Commenter: a.user,
	}

	_, err := a.client.SendComment(a.context, comment)
	if err != nil {
		log.Fatal(err)
	}
}
