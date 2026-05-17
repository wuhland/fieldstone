package workflows

import (
	"go.temporal.io/sdk/workflow"

	wftypes "github.com/fieldstone/fieldstone/internal/workflows"
)

// FOIARequestWorkflow is the durable execution for a FOIA request.
func FOIARequestWorkflow(ctx workflow.Context, input wftypes.WorkflowInput) error {
	completed := false

	if err := workflow.SetUpdateHandlerWithOptions(ctx,
		"validate-transition",
		func(ctx workflow.Context, req wftypes.TransitionRequest) error {
			if err := input.Config.ValidateTransition(req.From, req.To, req.Role); err != nil {
				return err
			}
			if input.Config.IsTerminal(req.To) {
				completed = true
			}
			return nil
		},
		workflow.UpdateHandlerOptions{},
	); err != nil {
		return err
	}

	// FOIA requests can be withdrawn by the resident who submitted them.
	withdrawCh := workflow.GetSignalChannel(ctx, "withdraw")
	workflow.Go(ctx, func(ctx workflow.Context) {
		var sig wftypes.WithdrawSignal
		withdrawCh.Receive(ctx, &sig)
		if input.ResidentID != nil && sig.ResidentID == *input.ResidentID {
			completed = true
		}
	})

	workflow.Await(ctx, func() bool { return completed })
	return nil
}
