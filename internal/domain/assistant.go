package domain

import (
	"fmt"
	"strings"
	"time"
)

// OperationSource tells who performed an operation.
type OperationSource string

const (
	// SourceDesktop is an operation performed in the desktop app. It is the
	// default when no source is recorded.
	SourceDesktop OperationSource = "desktop"
	// SourceAssistant is an operation performed through the assistant (MCP)
	// connection.
	SourceAssistant OperationSource = "assistant"
)

// Normalize returns SourceDesktop for the zero value and the value itself
// otherwise.
func (s OperationSource) Normalize() OperationSource {
	if s == "" {
		return SourceDesktop
	}
	return s
}

// AssistantOutcome is the result recorded for an assistant operation.
type AssistantOutcome string

const (
	OutcomeDone     AssistantOutcome = "done"
	OutcomeRejected AssistantOutcome = "rejected"
	OutcomeDeclined AssistantOutcome = "declined"
	OutcomeTimedOut AssistantOutcome = "timed_out"
	OutcomeRefused  AssistantOutcome = "refused"
)

// AssistantEntityKind names the kind of entity an assistant operation touched.
type AssistantEntityKind string

const (
	EntityCampaign        AssistantEntityKind = "campaign"
	EntityPlayerCharacter AssistantEntityKind = "player_character"
	EntityEncounter       AssistantEntityKind = "encounter"
	EntityCombatant       AssistantEntityKind = "combatant"
	EntityMonsterTemplate AssistantEntityKind = "monster_template"
)

const (
	// DefaultAssistantPort is the loopback port the assistant connection uses
	// until the GM chooses another one.
	DefaultAssistantPort = 47823
	minAssistantPort     = 1024
	maxAssistantPort     = 65535
)

// AccessSecret is the bearer secret assistant clients must present. Its
// String and GoString methods never reveal the value.
type AccessSecret string

func (AccessSecret) String() string { return "[redacted]" }

func (AccessSecret) GoString() string { return "[redacted]" }

// AssistantConnectionSettings is the GM's assistant connection choice.
type AssistantConnectionSettings struct {
	Enabled      bool
	Port         int
	AccessSecret AccessSecret
}

// ValidateAssistantPort checks that port is an unprivileged TCP port.
func ValidateAssistantPort(port int) error {
	if port < minAssistantPort || port > maxAssistantPort {
		return fmt.Errorf("port must be between %d and %d", minAssistantPort, maxAssistantPort)
	}
	return nil
}

// Validate checks the port range and that the access secret is not blank.
func (s AssistantConnectionSettings) Validate() error {
	if err := ValidateAssistantPort(s.Port); err != nil {
		return err
	}
	if strings.TrimSpace(string(s.AccessSecret)) == "" {
		return fmt.Errorf("access secret is required")
	}
	return nil
}

// AssistantActivityEntry is one record in the assistant activity log.
type AssistantActivityEntry struct {
	ID          string
	OccurredAt  time.Time
	Operation   string
	EntityKind  AssistantEntityKind
	EntityID    string
	EntityLabel string
	Outcome     AssistantOutcome
	Detail      string
}
