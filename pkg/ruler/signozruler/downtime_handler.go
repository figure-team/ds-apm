package signozruler

import (
	"context"
	"net/http"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/http/binding"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/gorilla/mux"
)

func (handler *handler) ListDowntimeSchedules(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	claims, err := authtypes.ClaimsFromContext(ctx)
	if err != nil {
		render.Error(rw, err)
		return
	}

	var params ruletypes.ListPlannedMaintenanceParams
	if err := binding.Query.BindQuery(req.URL.Query(), &params); err != nil {
		render.Error(rw, err)
		return
	}

	schedules, err := handler.ruler.MaintenanceStore().ListPlannedMaintenance(ctx, claims.OrgID)
	if err != nil {
		render.Error(rw, err)
		return
	}

	if params.Active != nil {
		activeSchedules := make([]*ruletypes.PlannedMaintenance, 0)
		for _, schedule := range schedules {
			now := time.Now().In(time.FixedZone(schedule.Schedule.Timezone, 0))
			if schedule.IsActive(now) == *params.Active {
				activeSchedules = append(activeSchedules, schedule)
			}
		}
		schedules = activeSchedules
	}

	if params.Recurring != nil {
		recurringSchedules := make([]*ruletypes.PlannedMaintenance, 0)
		for _, schedule := range schedules {
			if schedule.IsRecurring() == *params.Recurring {
				recurringSchedules = append(recurringSchedules, schedule)
			}
		}
		schedules = recurringSchedules
	}

	render.Success(rw, http.StatusOK, schedules)
}

func (handler *handler) GetDowntimeScheduleByID(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	id, err := valuer.NewUUID(mux.Vars(req)["id"])
	if err != nil {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "id is not a valid uuid-v7"))
		return
	}

	schedule, err := handler.ruler.MaintenanceStore().GetPlannedMaintenanceByID(ctx, id)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusOK, schedule)
}

func (handler *handler) CreateDowntimeSchedule(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	schedule := new(ruletypes.PostablePlannedMaintenance)
	if err := binding.JSON.BindBody(req.Body, schedule); err != nil {
		render.Error(rw, err)
		return
	}

	if err := schedule.Validate(); err != nil {
		render.Error(rw, err)
		return
	}

	created, err := handler.ruler.MaintenanceStore().CreatePlannedMaintenance(ctx, schedule)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusCreated, created)
}

func (handler *handler) UpdateDowntimeScheduleByID(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	id, err := valuer.NewUUID(mux.Vars(req)["id"])
	if err != nil {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "id is not a valid uuid-v7"))
		return
	}

	schedule := new(ruletypes.PostablePlannedMaintenance)
	if err := binding.JSON.BindBody(req.Body, schedule); err != nil {
		render.Error(rw, err)
		return
	}

	if err := schedule.Validate(); err != nil {
		render.Error(rw, err)
		return
	}

	err = handler.ruler.MaintenanceStore().UpdatePlannedMaintenance(ctx, schedule, id)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusNoContent, nil)
}

func (handler *handler) DeleteDowntimeScheduleByID(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	id, err := valuer.NewUUID(mux.Vars(req)["id"])
	if err != nil {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "id is not a valid uuid-v7"))
		return
	}

	err = handler.ruler.MaintenanceStore().DeletePlannedMaintenance(ctx, id)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusNoContent, nil)
}
