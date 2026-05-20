package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	wftypes "github.com/fieldstone/fieldstone/internal/workflows"
	workeractivities "github.com/fieldstone/fieldstone/services/workflow-worker/activities"
)

// PermitWorkflow is the durable execution for a single permit application.
// It validates every status transition via an Update handler, fires an auto-expiry
// timer when the permit reaches a status with auto_expire_after configured, and
// supports resident withdrawal via signal.
func PermitWorkflow(ctx workflow.Context, input wftypes.WorkflowInput) error {
	completed := false

	if err := workflow.SetUpdateHandlerWithOptions(ctx,
		"validate-transition",
		func(ctx workflow.Context, req wftypes.TransitionRequest) error {
			if err := input.Config.ValidateTransition(req.From, req.To, req.Role); err != nil {
				return err
			}
			if input.Config.IsTerminal(req.To) {
				completed = true
				return nil
			}
			if d := input.Config.AutoExpireAfterDuration(req.To); d > 0 {
				toStatus := req.To
				workflow.Go(ctx, func(timerCtx workflow.Context) {
					_ = workflow.Sleep(timerCtx, d)
					if completed {
						return
					}
					ao := workflow.WithActivityOptions(timerCtx, workflow.ActivityOptions{
						StartToCloseTimeout: 30 * time.Second,
						RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 5},
					})
					var a *workeractivities.PermitActivities
					_ = workflow.ExecuteActivity(ao, a.UpdatePermitStatus,
						workeractivities.UpdatePermitStatusParams{
							PermitID:  input.ResourceID,
							NewStatus: "expired",
							OldStatus: toStatus,
						}).Get(timerCtx, nil)
					completed = true
				})
			}
			return nil
		},
		workflow.UpdateHandlerOptions{},
	); err != nil {
		return err
	}

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
