package actions

import (
	"fmt"
	"slices"

	"github.com/axwell0/elmakina/engine/core/types"
)

// ActionRunner is the execution behavior every action implementation must satisfy.
type ActionRunner interface {
	Run(ctx *ActionContext) error
}

// ActionDefinition is the catalog entry for any action — main or counter.
// Counter actions are distinguished by a non-nil Counter facet.
type ActionDefinition struct {
	ID          types.ActionID
	Label       string
	Description string
	Payload     types.ActionPayload
	Role        types.Role
	Cost        int
	Runner      ActionRunner
	Eligibility []EligibilityCriterion
	Command     []CommandCriterion

	// Counter is set for reactive actions; nil for main actions.
	Counter *CounterFacet
}

// CounterFacet carries the fields that only counter actions need.
type CounterFacet struct {
	AllowedAgainst         []types.ActionID
	CancelsMain            bool
	MaxCounterDeclarations int
}

// IsCounter reports whether this definition is a reactive (counter) action.
func (d ActionDefinition) IsCounter() bool { return d.Counter != nil }

// IsMain reports whether this definition is a main action.
func (d ActionDefinition) IsMain() bool { return d.Counter == nil }

// ActionPresentation is the user-facing subset of a definition.
type ActionPresentation struct {
	Label       string
	Description string
	Payload     types.ActionPayload
}

// RoleDefinition is a derived view of actions that claim one role.
type RoleDefinition struct {
	ID             types.Role
	MainActions    []ActionDefinition
	CounterActions []ActionDefinition
}

// actions is the source of truth, declared in game-logical order.
var actions = []ActionDefinition{
	{
		ID:          types.Income,
		Label:       "Income",
		Description: "Take 1 coin safely.",
		Runner:      gainCoinsAction(1),
		Eligibility: []EligibilityCriterion{CoinEligibilityCriterion{MinCoins: 0}},
	},
	{
		ID:          types.ForeignAid,
		Label:       "Foreign Aid",
		Description: "Take 2 coins unless another player blocks it.",
		Runner:      gainCoinsAction(2),
		Eligibility: []EligibilityCriterion{CoinEligibilityCriterion{MinCoins: 0}},
	},
	{
		ID:          types.Business,
		Label:       "Businesswoman",
		Description: "Claim the Businesswoman and take a big payout unless the table taxes it down.",
		Role:        types.Businesswoman,
		Runner:      gainCoinsAction(4),
		Eligibility: []EligibilityCriterion{CoinEligibilityCriterion{MinCoins: 0}},
	},
	{
		ID:          types.Tax,
		Label:       "Tax",
		Description: "Claim the Tax Collector and take 3 coins.",
		Role:        types.TaxCollector,
		Runner:      actionFunc(runTax),
		Eligibility: []EligibilityCriterion{OpponentEligibilityCriterion{MinCoins: 7}},
		Command:     []CommandCriterion{TaxableOpponentCriterion{}},
	},
	{
		ID:          types.Investigate,
		Label:       "Investigate",
		Description: "Claim the Policewoman and target a player for the response chain.",
		Payload:     types.TargetPayload{},
		Role:        types.Policewoman,
		Runner:      actionFunc(runInvestigate),
		Eligibility: []EligibilityCriterion{OpponentEligibilityCriterion{}},
		Command:     []CommandCriterion{TargetAliveCriterion{}},
	},
	{
		ID:          types.Accuse,
		Label:       "Accuse",
		Description: "Claim the Colonel, target a player, and name the role you think they hold.",
		Payload:     types.AccusePayload{},
		Role:        types.Colonel,
		Cost:        4,
		Runner:      actionFunc(runAccuse),
		Eligibility: []EligibilityCriterion{
			CoinEligibilityCriterion{MinCoins: 4},
			OpponentEligibilityCriterion{},
		},
		Command: []CommandCriterion{TargetAliveCriterion{}},
	},
	{
		ID:          types.Assassinate,
		Label:       "Assassinate",
		Description: "Claim the Terrorist, pay 3 coins, and try to remove a player's card.",
		Payload:     types.TargetPayload{},
		Role:        types.Terrorist,
		Cost:        3,
		Runner:      discardAction("assassinate"),
		Eligibility: []EligibilityCriterion{
			CoinEligibilityCriterion{MinCoins: 3},
			OpponentEligibilityCriterion{},
		},
		Command: []CommandCriterion{TargetAliveCriterion{}},
	},
	{
		ID:          types.Steal,
		Label:       "Steal",
		Description: "Claim the Thief and take coins from another player.",
		Payload:     types.TargetPayload{},
		Role:        types.Thief,
		Runner:      actionFunc(runSteal),
		Eligibility: []EligibilityCriterion{OpponentEligibilityCriterion{MinCoins: 2}},
		Command: []CommandCriterion{
			TargetAliveCriterion{},
			TargetCoinCriterion{MinCoins: 2},
		},
	},
	{
		ID:          types.Exchange,
		Label:       "Exchange",
		Description: "Claim the Politician and draw into an exchange step.",
		Role:        types.Politician,
		Runner:      ExchangeAction{},
		Eligibility: []EligibilityCriterion{CoinEligibilityCriterion{MinCoins: 0}},
	},
	{
		ID:          types.Coup,
		Label:       "Coup",
		Description: "Spend 7 coins to force a player to lose influence.",
		Payload:     types.TargetPayload{},
		Cost:        7,
		Runner:      discardAction("coup"),
		Eligibility: []EligibilityCriterion{
			CoinEligibilityCriterion{MinCoins: 7},
			OpponentEligibilityCriterion{},
		},
		Command: []CommandCriterion{TargetAliveCriterion{}},
	},
	{
		ID:          types.Pass,
		Label:       "Pass",
		Description: "Do nothing and yield the moment.",
		Runner:      noOpAction(),
	},

	// Counter actions follow.
	{
		ID:          types.TaxBusinessWoman,
		Label:       "Tax Businesswoman",
		Description: "Reduce a Businesswoman payout instead of canceling it outright.",
		Role:        types.TaxCollector,
		Runner:      actionFunc(runTaxBusinessWoman),
		Counter: &CounterFacet{
			AllowedAgainst:         []types.ActionID{types.Business},
			CancelsMain:            false,
			MaxCounterDeclarations: 4,
		},
	},
	{
		ID:          types.BlockSteal,
		Label:       "Block Steal",
		Description: "Stop a steal attempt.",
		Role:        types.Thief,
		Runner:      noOpAction(),
		Counter: &CounterFacet{
			AllowedAgainst: []types.ActionID{types.Steal},
			CancelsMain:    true,
		},
	},
	{
		ID:          types.BlockTerrorist,
		Label:       "Block Terrorist",
		Description: "Block an assassination attempt.",
		Role:        types.Colonel,
		Runner:      noOpAction(),
		Counter: &CounterFacet{
			AllowedAgainst: []types.ActionID{types.Assassinate},
			CancelsMain:    true,
		},
	},
	{
		ID:          types.BlockPolice,
		Label:       "Block Investigate",
		Description: "Stop an investigate attempt from going through.",
		Role:        types.Policewoman,
		Runner:      noOpAction(),
		Counter: &CounterFacet{
			AllowedAgainst: []types.ActionID{types.Investigate},
			CancelsMain:    true,
		},
	},
	{
		ID:          types.BlockForeignAid,
		Label:       "Block Foreign Aid",
		Description: "Stop a foreign aid attempt from landing.",
		Role:        types.TaxCollector,
		Runner:      noOpAction(),
		Counter: &CounterFacet{
			AllowedAgainst: []types.ActionID{types.ForeignAid},
			CancelsMain:    true,
		},
	},
	{
		ID:          types.Escape,
		Label:       "Escape",
		Description: "Pay heavily to avoid a coup or assassination.",
		Cost:        9,
		Runner:      noOpAction(),
		Eligibility: []EligibilityCriterion{
			ResponderCoinEligibilityCriterion{MinCoins: 9},
		},
		Counter: &CounterFacet{
			AllowedAgainst: []types.ActionID{types.Coup, types.Assassinate},
			CancelsMain:    true,
		},
	},
}

var (
	actionByID     map[types.ActionID]ActionDefinition
	actionOrder    []types.ActionID
	countersByMain map[types.ActionID][]ActionDefinition
	mainsByRole    map[types.Role][]ActionDefinition
	countersByRole map[types.Role][]ActionDefinition
)

func init() {
	actionByID = make(map[types.ActionID]ActionDefinition, len(actions))
	actionOrder = make([]types.ActionID, 0, len(actions))
	countersByMain = make(map[types.ActionID][]ActionDefinition)
	mainsByRole = make(map[types.Role][]ActionDefinition)
	countersByRole = make(map[types.Role][]ActionDefinition)

	for _, def := range actions {
		if _, dup := actionByID[def.ID]; dup {
			panic(fmt.Sprintf("actions: duplicate action id %q", def.ID))
		}
		if def.IsCounter() && len(def.Counter.AllowedAgainst) == 0 {
			panic(fmt.Sprintf("actions: counter %q has no AllowedAgainst", def.ID))
		}
		actionByID[def.ID] = def
		actionOrder = append(actionOrder, def.ID)

		switch {
		case def.IsCounter():
			for _, main := range def.Counter.AllowedAgainst {
				countersByMain[main] = append(countersByMain[main], def)
			}
			if def.Role != "" {
				countersByRole[def.Role] = append(countersByRole[def.Role], def)
			}
		case def.Role != "":
			mainsByRole[def.Role] = append(mainsByRole[def.Role], def)
		}
	}
}

// Action returns the definition for any registered action id.
func Action(id types.ActionID) (ActionDefinition, bool) {
	def, ok := actionByID[id]
	return def, ok
}

// MainAction returns id iff it names a main action.
func MainAction(id types.ActionID) (ActionDefinition, bool) {
	def, ok := actionByID[id]
	if !ok || def.IsCounter() {
		return ActionDefinition{}, false
	}
	return def, true
}

// CounterAction returns id iff it names a counter action.
func CounterAction(id types.ActionID) (ActionDefinition, bool) {
	def, ok := actionByID[id]
	if !ok || !def.IsCounter() {
		return ActionDefinition{}, false
	}
	return def, true
}

// MainActions returns the offerable main action ids in declaration order.
// types.Pass is registered but never offered.
func MainActions() []types.ActionID {
	out := make([]types.ActionID, 0, len(actionOrder))
	for _, id := range actionOrder {
		def := actionByID[id]
		if def.IsMain() && def.ID != types.Pass {
			out = append(out, id)
		}
	}
	return out
}

// CounterActions returns counter action ids in declaration order.
func CounterActions() []types.ActionID {
	out := make([]types.ActionID, 0, len(actionOrder))
	for _, id := range actionOrder {
		if def := actionByID[id]; def.IsCounter() {
			out = append(out, id)
		}
	}
	return out
}

// CountersForMain returns all counters declared as AllowedAgainst the given main id.
func CountersForMain(mainActionID types.ActionID) []ActionDefinition {
	return countersByMain[mainActionID]
}

// Role returns the derived view of mains and counters claiming roleID.
func Role(roleID types.Role) (RoleDefinition, bool) {
	if !slices.Contains(types.Roles, roleID) {
		return RoleDefinition{}, false
	}
	return RoleDefinition{
		ID:             roleID,
		MainActions:    mainsByRole[roleID],
		CounterActions: countersByRole[roleID],
	}, true
}

// Roles returns every role definition in types.Roles order.
func Roles() []RoleDefinition {
	roles := make([]RoleDefinition, 0, len(types.Roles))
	for _, roleID := range types.Roles {
		role, _ := Role(roleID)
		roles = append(roles, role)
	}
	return roles
}

// ActionPresentationByID returns the user-facing subset for any action.
func ActionPresentationByID(id types.ActionID) (ActionPresentation, bool) {
	def, ok := actionByID[id]
	if !ok || def.Label == "" || def.Description == "" {
		return ActionPresentation{}, false
	}
	return ActionPresentation{
		Label:       def.Label,
		Description: def.Description,
		Payload:     def.Payload,
	}, true
}

// ActionLabel returns the human label for id, or "" if unknown.
func ActionLabel(id types.ActionID) string {
	if presentation, ok := ActionPresentationByID(id); ok {
		return presentation.Label
	}
	return ""
}
