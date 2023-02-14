package main

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/samber/lo"
	"github.com/t-kuni/grpc-example/grpc/chat"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"strings"
)

var (
	client chat.ChatClient
	ctx    context.Context
	user   *chat.User
	state  *chat.State
	prog   *tea.Program
)

type (
	errMsg          error
	stateUpdatedMsg struct{}
)

func main() {
	port := 30000
	ctx = context.Background()

	fmt.Print("\033[H\033[2J") // コンソールをクリア

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	serverAddr := fmt.Sprintf("localhost:%d", port)
	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client = chat.NewChatClient(conn)

	var name string
	fmt.Print("Your Name: ")
	_, err = fmt.Scan(&name)
	if err != nil {
		log.Fatal(err)
	}

	profile := &chat.Profile{
		Name:   name,
		Age:    10,
		Gender: chat.Gender_GENDER_MAN,
	}
	user, err = client.Join(ctx, profile)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("\033[H\033[2J") // コンソールをクリア

	prog = tea.NewProgram(initialModel())

	go watchState()

	if _, err := prog.Run(); err != nil {
		log.Fatal(err)
	}
}

func watchState() {
	stream, err := client.WatchState(ctx, &empty.Empty{})
	if err != nil {
		log.Fatal(err)
	}
	for {
		state, err = stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		prog.Send(stateUpdatedMsg{})
	}
}

type model struct {
	nameView       viewport.Model
	joinedUserView viewport.Model
	commentView    viewport.Model
	textInput      textinput.Model
	senderStyle    lipgloss.Style
	err            error
}

func initialModel() model {
	width := 50

	ti := textinput.New()
	ti.Placeholder = "Comment"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = width

	nameView := viewport.New(width, 1)
	nameView.SetContent(`Your Name: ` + user.Profile.Name)

	joinedUserView := viewport.New(width, 1)
	joinedUserView.SetContent(`Members: `)

	commentView := viewport.New(width, 10)
	commentView.SetContent(``)

	return model{
		nameView:       nameView,
		joinedUserView: joinedUserView,
		textInput:      ti,
		commentView:    commentView,
		senderStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		err:            nil,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textInput, tiCmd = m.textInput.Update(msg)
	m.commentView, vpCmd = m.commentView.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			fmt.Println(m.textInput.Value())
			return m, tea.Quit
		case tea.KeyEnter:
			commentBody := m.textInput.Value()
			m.textInput.Reset()

			comment := &chat.Comment{
				Body:      commentBody,
				Commenter: user,
			}

			_, err := client.SendComment(ctx, comment)
			if err != nil {
				log.Fatal(err)
			}
		}
	case stateUpdatedMsg:
		joinedUserTexts := lo.Map[*chat.User, string](state.JoinedUsers, func(item *chat.User, index int) string {
			return item.Profile.Name
		})
		m.joinedUserView.SetContent("Members: " + strings.Join(joinedUserTexts, ", "))

		commentTexts := lo.Map[*chat.Comment, string](state.LatestComments, func(comment *chat.Comment, index int) string {
			name := comment.Commenter.Profile.Name
			return m.senderStyle.Render(name+": ") + comment.Body
		})
		m.commentView.SetContent(strings.Join(commentTexts, "\n"))

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m model) View() string {
	return fmt.Sprintf(
		"%s\n%s\n%s\n%s",
		m.nameView.View(),
		m.joinedUserView.View(),
		m.commentView.View(),
		m.textInput.View(),
	)
}
