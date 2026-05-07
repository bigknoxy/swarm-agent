package orchestrator

// MockCLI is a mock CLI for testing
type MockCLI struct{}

func (m *MockCLI) PrintStart(goal string) {}
func (m *MockCLI) PrintThought(thought string) {}
func (m *MockCLI) PrintAction(action string) {}
func (m *MockCLI) PrintObservation(obs string) {}
func (m *MockCLI) PrintSuccess(msg string) {}
func (m *MockCLI) PrintFailure(msg string) {}
func (m *MockCLI) PrintInfo(msg string) {}
func (m *MockCLI) PrintVerbose(label, content string) {}
func (m *MockCLI) PrintJudgeVerdict(verdict string) {}
func (m *MockCLI) NewProgressBar(max int) interface{} { return nil }
