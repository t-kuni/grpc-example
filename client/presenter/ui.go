package presenter

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"
	"github.com/t-kuni/grpc-example/grpc/chat"
	"log"
	"strings"
)

type (
	ErrMsg          error
	StateUpdatedMsg struct {
		State *chat.State
	}

	UI struct {
		ViewModel *ViewModel
		prog      *tea.Program
	}
)

func NewUI(userName string) *UI {
	vm := newViewModel(userName)

	prog := tea.NewProgram(vm)

	return &UI{
		ViewModel: vm,
		prog:      prog,
	}
}

func (u UI) Start() error {
	_, err := u.prog.Run()
	return err
}

func (u UI) Send(msg tea.Msg) {
	u.prog.Send(msg)
}

type ViewModel struct {
	nameView           viewport.Model
	joinedUserView     viewport.Model
	commentView        viewport.Model
	textInput          textinput.Model
	senderStyles       []lipgloss.Style
	systemCommentStyle lipgloss.Style
	err                error

	QuitHandler        func()
	SendCommentHandler func(comment string)
}

func newViewModel(userName string) *ViewModel {
	width := 50

	ti := textinput.New()
	ti.Placeholder = "Comment"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = width

	nameView := viewport.New(width, 1)
	nameView.SetContent(`Your Name: ` + userName)

	joinedUserView := viewport.New(width, 1)
	joinedUserView.SetContent(`Members: `)

	commentView := viewport.New(width, 10)
	commentView.SetContent(``)

	return &ViewModel{
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

func (m ViewModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m ViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			m.QuitHandler()
			return m, tea.Quit
		case tea.KeyEnter:
			commentBody := strings.TrimSpace(m.textInput.Value())
			if commentBody == "" {
				break
			}
			m.textInput.Reset()
			m.SendCommentHandler(commentBody)
		}
	case StateUpdatedMsg:
		joinedUserTexts := lo.Map[*chat.User, string](msg.State.JoinedUsers, func(item *chat.User, index int) string {
			style := m.senderStyles[item.Color]
			return style.Render(item.Profile.Name)
		})
		m.joinedUserView.SetContent("Members: " + strings.Join(joinedUserTexts, ", "))

		commentTexts := lo.Map[*chat.Comment, string](msg.State.LatestComments, func(comment *chat.Comment, index int) string {
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
	case ErrMsg:
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m ViewModel) View() string {
	return fmt.Sprintf(
		"%s\n%s\n%s\n%s",
		m.nameView.View(),
		m.joinedUserView.View(),
		m.commentView.View(),
		m.textInput.View(),
	)
}

func ClearConsole() {
	fmt.Print("\033[H\033[2J")
}

func ScanUserName() string {
	var name string
	fmt.Print("Your Name: ")
	_, err := fmt.Scan(&name)
	if err != nil {
		log.Fatal(err)
	}
	return name
}
