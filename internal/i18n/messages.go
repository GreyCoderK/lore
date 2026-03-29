// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package i18n

// Messages is the root catalog containing all user-facing strings
// organized by functional domain. Each sub-struct is defined in its
// own file (messages_cmd.go, etc.) for maintainability.
type Messages struct {
	Cmd        CmdMessages
	Workflow   WorkflowMessages
	UI         UIMessages
	Angela     AngelaMessages
	Engagement EngagementMessages
	Decision   DecisionMessages
	Shared     SharedMessages
}
