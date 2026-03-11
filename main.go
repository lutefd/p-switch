package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

//go:embed config.example.yaml
var exampleConfig string

type Profile struct {
	Name            string `yaml:"name"`
	Email           string `yaml:"email"`
	SSHIdentityFile string `yaml:"sshIdentityFile"`
	GPGKey          string `yaml:"gpgKey"`
}

type Config struct {
	Profiles      map[string]Profile `yaml:"profiles"`
	SSHConfigPath string             `yaml:"sshConfigPath"`
}

type model struct {
	choices  []string
	cursor   int
	selected string
	config   Config
	err      error
	success  bool
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			MarginTop(1).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true).
			PaddingLeft(2)

	normalStyle = lipgloss.NewStyle().
			PaddingLeft(4)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true).
			MarginTop(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87")).
			Bold(true).
			MarginTop(1)
)

func loadConfig() (Config, error) {
	var config Config

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return config, err
	}

	configDir := filepath.Join(homeDir, ".config", "p-switch")
	configPath := filepath.Join(configDir, "config.yaml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return config, fmt.Errorf("failed to create config directory: %w", err)
		}

		if err := os.WriteFile(configPath, []byte(exampleConfig), 0644); err != nil {
			return config, fmt.Errorf("failed to write example config: %w", err)
		}

		return config, fmt.Errorf("First run! Created config at: %s\n\nPlease edit it with your details:\n- Add your profile names and emails\n- Set your SSH identity file paths\n\nThen run p-switch again!", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return config, fmt.Errorf("failed to read config: %w", err)
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}

func initialModel() model {
	config, err := loadConfig()
	if err != nil {
		return model{err: err}
	}

	choices := make([]string, 0, len(config.Profiles))
	for name := range config.Profiles {
		choices = append(choices, name)
	}

	return model{
		choices: choices,
		config:  config,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.success {
		switch msg.(type) {
		case tea.KeyMsg:
			return m, tea.Quit
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "enter", " ":
			m.selected = m.choices[m.cursor]
			err := switchProfile(m.selected, m.config)
			if err != nil {
				m.err = err
			} else {
				m.success = true
			}
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return errorStyle.Render("Error: " + m.err.Error() + "\n")
	}

	if m.success {
		profile := m.config.Profiles[m.selected]
		gpgInfo := "none"
		if profile.GPGKey != "" {
			gpgInfo = profile.GPGKey
		}
		return successStyle.Render(fmt.Sprintf("Switched to %s profile!\n\n"+
			"Email: %s\n"+
			"SSH Key: %s\n"+
			"GPG Key: %s\n\n"+
			"Press any key to exit...\n",
			m.selected, profile.Email, profile.SSHIdentityFile, gpgInfo))
	}

	s := titleStyle.Render("Git Profile Switcher") + "\n\n"
	s += "Select a profile:\n\n"

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
			s += selectedStyle.Render(fmt.Sprintf("%s %s", cursor, choice)) + "\n"
		} else {
			s += normalStyle.Render(choice) + "\n"
		}
	}

	s += "\n" + lipgloss.NewStyle().Faint(true).Render("(↑/↓ to navigate, enter to select, q to quit)")

	return s
}

func switchProfile(profileName string, config Config) error {
	profile, ok := config.Profiles[profileName]
	if !ok {
		return fmt.Errorf("profile %s not found", profileName)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	sshConfigPath := strings.Replace(config.SSHConfigPath, "~", homeDir, 1)

	sshData, err := os.ReadFile(sshConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read SSH config: %w", err)
	}

	lines := strings.Split(string(sshData), "\n")
	var newLines []string
	skipNextLines := 0

	for i, line := range lines {
		if skipNextLines > 0 {
			skipNextLines--
			continue
		}

		if strings.HasPrefix(strings.TrimSpace(line), "Host github.com") &&
			!strings.Contains(line, "-personal") &&
			!strings.Contains(line, "-emu") {
			newLines = append(newLines, line)
			for j := i + 1; j < len(lines) && j < i+10; j++ {
				nextLine := lines[j]
				if strings.HasPrefix(strings.TrimSpace(nextLine), "Host ") {
					break
				}
				if strings.Contains(nextLine, "IdentityFile") {
					newLines = append(newLines, strings.Split(nextLine, "IdentityFile")[0]+"IdentityFile "+profile.SSHIdentityFile)
					skipNextLines++
					continue
				}
				newLines = append(newLines, nextLine)
				skipNextLines++
			}
			continue
		}

		if skipNextLines == 0 {
			newLines = append(newLines, line)
		}
	}

	err = os.WriteFile(sshConfigPath, []byte(strings.Join(newLines, "\n")), 0600)
	if err != nil {
		return fmt.Errorf("failed to write SSH config: %w", err)
	}

	cmd := exec.Command("git", "config", "--global", "user.email", profile.Email)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set git email: %w", err)
	}

	cmd = exec.Command("git", "config", "--global", "user.name", profile.Name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set git name: %w", err)
	}

	if profile.GPGKey != "" {
		cmd = exec.Command("git", "config", "--global", "user.signingkey", profile.GPGKey)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set gpg signing key: %w", err)
		}

		cmd = exec.Command("git", "config", "--global", "commit.gpgsign", "true")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to enable commit signing: %w", err)
		}
	} else {
		cmd = exec.Command("git", "config", "--global", "--unset", "user.signingkey")
		_ = cmd.Run() // ignore error if key wasn't set

		cmd = exec.Command("git", "config", "--global", "commit.gpgsign", "false")
		_ = cmd.Run()
	}

	return nil
}

func main() {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".config", "p-switch", "config.yaml")

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--config", "-c":
			fmt.Println("Config location:", configPath)
			return
		case "--edit", "-e":
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vim"
			}
			cmd := exec.Command(editor, configPath)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Printf("Error opening editor: %v\n", err)
				os.Exit(1)
			}
			return
		case "--help", "-h":
			fmt.Println("p-switch - Git Profile Switcher")
			fmt.Println()
			fmt.Println("Usage:")
			fmt.Println("  p-switch           Run the interactive profile switcher")
			fmt.Println("  p-switch --config  Show config file location")
			fmt.Println("  p-switch --edit    Edit config file")
			fmt.Println("  p-switch --help    Show this help message")
			return
		}
	}

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
