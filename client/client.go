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
	"os"
	"os/signal"
	"strings"
	"syscall"
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

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		client.Leave(ctx, user)
		os.Exit(1)
	}()

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
	nameView           viewport.Model
	joinedUserView     viewport.Model
	commentView        viewport.Model
	textInput          textinput.Model
	senderStyles       []lipgloss.Style
	systemCommentStyle lipgloss.Style
	err                error
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
		senderStyles: []lipgloss.Style{
			lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
			lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
			lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
			lipgloss.NewStyle().Foreground(lipgloss.Color("4")),
			lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
			lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
		},
		systemCommentStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")),
		err:                nil,
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
			if user != nil {
				client.Leave(ctx, user)
			}
			return m, tea.Quit
		case tea.KeyEnter:
			commentBody := strings.TrimSpace(m.textInput.Value())
			if commentBody == "" {
				break
			}

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
			style := m.senderStyles[item.Color]
			return style.Render(item.Profile.Name)
		})
		m.joinedUserView.SetContent("Members: " + strings.Join(joinedUserTexts, ", "))

		commentTexts := lo.Map[*chat.Comment, string](state.LatestComments, func(comment *chat.Comment, index int) string {
			if comment.IsSystemComment {
				return m.systemCommentStyle.Render(comment.Body)
			} else {
				name := comment.Commenter.Profile.Name
				style := m.senderStyles[comment.Commenter.Color]
				return style.Render(name+": ") + comment.Body
			}
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
