package llm

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

var actionabilityTemplate = prompt.FromMessages(schema.GoTemplate,
	schema.SystemMessage(
		`You are a strict actionability classifier for a personal task-tracking app.
You are deciding for one specific person: the OWNER.

OWNER IDENTITY
- Owner emails (any address listed here belongs to the owner): {{.owner_emails}}
- Owner name: {{.user_name}}

Treat every listed email as a valid identity for the owner — the owner may receive mail on any of them, and may send mail from any of them. If owner name is blank, ignore name-based checks and rely on emails and explicit @-tags only. Events may be in any language; judge semantics, not surface strings.

DEFAULT STANCE
Default to actionable=false. Only flip to true when at least one event contains an unambiguous, direct, personal ask addressed to the owner. When in doubt, reject. Missing a real task is far better than creating noise. Weak signals (topic relevance, owner being a passive recipient, presence in a CC/TO list of many, generic greetings) NEVER override the default.

HARD REJECT (always actionable=false, regardless of other signals)
- Telegram group chats (TEXT contains type=group) where no message names the owner, tags the owner, or addresses the owner in second person. Marketplace, classifieds, announcements, polls, generic "кто подскажет / does anyone know" broadcasts in groups: reject.
- Newsletters, digests, mass emails, marketing, promotions, "no-reply" senders.
- Receipts, order confirmations, shipping updates, statements, invoices already paid, bank alerts that need no action.
- OTP, 2FA, verification codes, password resets, login alerts.
- Bot notifications, CI/build/alert system messages, automated reminders the owner already set elsewhere.
- Social-media notifications, likes, follows, reaction pings.
- Events authored BY the owner (SENDER email matches any owner email) with no incoming reply requiring action.
- Forwarded / FYI content with no explicit request.

ACCEPT (actionable=true) ONLY IF AT LEAST ONE applies
- Direct Telegram private message, or email whose TO is a small list including any owner email, with a concrete request or commitment.
- Telegram group message that names the owner (owner name, owner's first name as a vocative, or @-mention) AND contains a concrete ask.
- Google Calendar invite where the owner is invitee or organizer and the event is upcoming.
- Email where body or subject directly addresses the owner by name or role AND requests a reply, decision, attendance, document, or deadline-bound action.

OUTPUT
Return valid JSON only, no prose, no code fences:
{
  "actionable": <bool>,
  "reason": "<one to two sentences citing the specific signal: which event, which sender, which phrase, addressing pattern, or absence-of-addressing led to the decision>"
}

The reason field is ALWAYS mandatory, regardless of the value of actionable. Cite the specific signal that drove the decision — for rejects, the concrete reject criterion that fired (e.g. "telegram group marketplace broadcast, no mention of any owner email or owner name"); for accepts, the concrete accept criterion (e.g. "email TO an owner address with explicit deadline-bound ask to review PR #47 from sender@example.com"). Do not penalize delivery to any of the listed owner emails — all of them are the owner.`),
	schema.UserMessage("Events:\n{{.events}}"),
)

func NewClusterActionabilityService(
	model model.BaseChatModel,
	logger log.Logger,
) *clusterActionabilityService {
	return &clusterActionabilityService{
		model:  model,
		logger: logger,
	}
}

type clusterActionabilityService struct {
	model  model.BaseChatModel
	logger log.Logger
}

func (s *clusterActionabilityService) GetTaskGenerationDecision(
	ctx context.Context,
	events []domain.Event,
	profile domain.UserProfile,
) (domain.TaskGenerationDecision, error) {
	eventsText, err := formatEvents(events)
	if err != nil {
		return domain.TaskGenerationDecision{}, err
	}

	messages, err := actionabilityTemplate.Format(ctx, map[string]any{
		"events":       eventsText,
		"owner_emails": formatOwnerEmails(profile),
		"user_name":    profile.Name,
	})
	if err != nil {
		return domain.TaskGenerationDecision{}, errors.WrapFail(err, "format messages for actionability llm")
	}

	return generateAndParseWithRetry(ctx, s.logger, s.model, messages, userIDFromEvents(events), parseActionabilityResponse)
}

func parseActionabilityResponse(responseText string) (domain.TaskGenerationDecision, error) {
	jsonText := extractJSON(responseText)
	if jsonText == "" {
		return domain.TaskGenerationDecision{}, errors.Errorf("no valid JSON found in response %v", errors.Token("response", responseText))
	}

	var llmResp llmActionabilityResponse
	if err := json.Unmarshal([]byte(jsonText), &llmResp); err != nil {
		return domain.TaskGenerationDecision{}, errors.WrapFailf(err, "unmarshal actionability response: %s", jsonText)
	}

	llmResp.Reason = strings.TrimSpace(llmResp.Reason)
	if llmResp.Reason == "" {
		return domain.TaskGenerationDecision{}, errors.Errorf("reason is required for every classification decision")
	}

	reason := llmResp.Reason
	return domain.TaskGenerationDecision{
		ShouldGenerate: llmResp.Actionable,
		Reason:         &reason,
	}, nil
}

type llmActionabilityResponse struct {
	Actionable bool   `json:"actionable"`
	Reason     string `json:"reason"`
}

func formatOwnerEmails(profile domain.UserProfile) string {
	seen := make(map[string]struct{}, 1+len(profile.ConnectedEmails))
	emails := make([]string, 0, 1+len(profile.ConnectedEmails))
	add := func(raw string) {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		emails = append(emails, trimmed)
	}
	add(profile.Email)
	for _, email := range profile.ConnectedEmails {
		add(email)
	}
	if len(emails) == 0 {
		return "(none)"
	}
	return strings.Join(emails, ", ")
}
