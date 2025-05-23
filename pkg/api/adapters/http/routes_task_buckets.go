package http

import (
	"encoding/json"
	"net/http"

	"github.com/eser/ajan/httpfx"
	"github.com/eser/bfo/pkg/api/adapters/appcontext"
	"github.com/eser/bfo/pkg/api/business/tasks"
)

func RegisterHttpRoutesForTaskBuckets( //nolint:funlen
	routes *httpfx.Router,
	appContext *appcontext.AppContext,
) {
	routes.
		Route(
			"PUT /api/{bucket}/schema",
			func(ctx *httpfx.Context) httpfx.Result {
				// get variables from path
				bucketParam := ctx.Request.PathValue("bucket")

				// get body
				var taskBucket tasks.TaskBucket
				err := json.NewDecoder(ctx.Request.Body).Decode(&taskBucket)
				if err != nil {
					return ctx.Results.Error(http.StatusInternalServerError, []byte(err.Error()))
				}

				// override task bucket id
				taskBucket.Id = bucketParam

				err = appContext.Tasks.UpdateTaskBucket(ctx.Request.Context(), &taskBucket)

				if err != nil {
					return ctx.Results.Error(http.StatusInternalServerError, []byte(err.Error()))
				}

				return ctx.Results.Ok()
			},
		).
		HasSummary("Set task bucket's schema").
		HasDescription("Set the schema for a task bucket.").
		HasResponse(http.StatusOK)
}
