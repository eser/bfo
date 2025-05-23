package http

import (
	"encoding/json"
	"net/http"

	"github.com/eser/ajan/httpfx"
	"github.com/eser/bfo/pkg/api/adapters/appcontext"
	"github.com/eser/bfo/pkg/api/business/tasks"
)

func RegisterHttpRoutesForTasks( //nolint:funlen
	routes *httpfx.Router,
	appContext *appcontext.AppContext,
) {
	routes.
		Route(
			"PUT /api/{bucket}/tasks/{id...}",
			func(ctx *httpfx.Context) httpfx.Result {
				// get variables from path
				bucketParam := ctx.Request.PathValue("bucket")
				idParam := ctx.Request.PathValue("id")

				// get body
				var task tasks.Task
				err := json.NewDecoder(ctx.Request.Body).Decode(&task)
				if err != nil {
					return ctx.Results.Error(http.StatusInternalServerError, []byte(err.Error()))
				}

				// override task id and bucket id
				task.Id = idParam
				task.BucketId = bucketParam

				err = appContext.Tasks.DispatchTask(ctx.Request.Context(), &task)

				if err != nil {
					return ctx.Results.Error(http.StatusInternalServerError, []byte(err.Error()))
				}

				return ctx.Results.Ok()
			},
		).
		HasSummary("Dispatch task").
		HasDescription("Dispatch a task to the task service.").
		HasResponse(http.StatusOK)
}
