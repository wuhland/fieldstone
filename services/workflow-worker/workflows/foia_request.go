package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	wftypes "github.com/fieldstone/fieldstone/internal/workflows"
	workeractivities "github.com/fieldstone/fieldstone/services/workflow-worker/activities"
)

// FOIARequestWorkflow is the durable execution for a FOIA request.
// It validates transitions, supports resident withdrawal, and fires a deadline
// notification activity if a due date was provided at workflow start.
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

	// If a deadline was provided, fire a notification activity when it passes.
	if input.Deadline != nil {
		deadline := *input.Deadline
		workflow.Go(ctx, func(timerCtx workflow.Context) {
			sleepDuration := deadline.Sub(workflow.Now(timerCtx))
			if sleepDuration <= 0 {
				return
			}
			_ = workflow.Sleep(timerCtx, sleepDuration)
			if completed {
				return
			}
			ao := workflow.WithActivityOptions(timerCtx, workflow.ActivityOptions{
				StartToCloseTimeout: 30 * time.Second,
				RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 5},
			})
			var a *workeractivities.RecordsActivities
			_ = workflow.ExecuteActivity(ao, a.NotifyDeadlineExceeded,
				workeractivities.DeadlineExceededParams{RequestID: input.ResourceID},
			).Get(timerCtx, nil)
		})
	}

	workflow.Await(ctx, func() bool { return completed })
	return nil
}
