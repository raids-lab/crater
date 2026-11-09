// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package extender

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	. "github.com/bytedance/mockey"
	"github.com/go-logr/logr/funcr"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/pkg/utils"
)

const (
	enqueuePath = "/" + JobEnqueueableVerb
	closePath   = "/" + OnSessionCloseVerb
)

// staleServer returns a server whose round is older than sessionStaleAfter, with the round rebuild
// dependencies stubbed to the given outcomes.
func staleServer(cfg *model.SchedulerExtenderConfig, cfgErr, quotaErr error) *Server {
	s := seededServer(newSnapshot())
	s.session.builtAt = fixedNow.Add(-sessionStaleAfter - time.Second)
	s.configService = &service.ConfigService{}
	s.quotaService = &service.QueueQuotaService{}
	Mock((*service.ConfigService).GetSchedulerExtenderConfig).Return(cfg, cfgErr).Build()
	Mock((*service.QueueQuotaService).LoadQuotaSet).Return(&service.QueueQuotaSet{Enabled: true}, quotaErr).Build()
	return s
}

// captureLog routes the server's log lines into the returned slice.
func captureLog(s *Server) *[]string {
	lines := &[]string{}
	s.logger = funcr.New(func(_, args string) { *lines = append(*lines, args) }, funcr.Options{})
	return lines
}

func enqueueRequest(body []byte) *http.Request {
	return httptest.NewRequest(http.MethodPost, enqueuePath, bytes.NewReader(body))
}

func TestCurrentSession(t *testing.T) {
	PatchConvey("reuses the round in progress", t, func() {
		s := seededServer(newSnapshot())
		s.configService = nil
		state, err := s.currentSession(t.Context())
		So(err, ShouldBeNil)
		So(state, ShouldEqual, s.session)
	})

	PatchConvey("rebuilds after the staleness window and sweeps the ledger", t, func() {
		s := staleServer(&model.SchedulerExtenderConfig{SchedulerExtenderEnabled: true, QueueQuotaEnabled: true}, nil, nil)
		previous := s.session
		fresh := newSnapshot(newView(jobA, rl("cpu", "1")))
		build := Mock((*Server).buildSnapshot).Return(fresh, nil).Build()
		s.accumulator.reserve(newView(jobB, rl("cpu", "1")))

		state, err := s.currentSession(t.Context())
		So(err, ShouldBeNil)
		So(state, ShouldNotEqual, previous)
		So(state, ShouldEqual, s.session)
		So(state.snap, ShouldEqual, fresh)
		So(state.settings.quotas.Enabled, ShouldBeTrue)
		So(state.builtAt, ShouldHappenAfter, previous.builtAt)
		So(build.MockTimes(), ShouldEqual, 1)
		So(s.accumulator.entries, ShouldNotContainKey, jobB)
	})

	PatchConvey("keeps the round while close notifications arrive", t, func() {
		s := staleServer(&model.SchedulerExtenderConfig{SchedulerExtenderEnabled: true}, nil, nil)
		s.lastCloseAt = utils.GetLocalTime()
		previous := s.session
		build := Mock((*Server).buildSnapshot).Return(newSnapshot(), nil).Build()
		s.accumulator.reserve(newView(jobB, rl("cpu", "1")))

		state, err := s.currentSession(t.Context())
		So(err, ShouldBeNil)
		So(state, ShouldEqual, previous)
		So(build.MockTimes(), ShouldEqual, 0)
		So(s.accumulator.entries, ShouldContainKey, jobB)
	})

	PatchConvey("rebuilds once close notifications stop", t, func() {
		s := staleServer(&model.SchedulerExtenderConfig{SchedulerExtenderEnabled: true}, nil, nil)
		s.lastCloseAt = fixedNow
		previous := s.session
		build := Mock((*Server).buildSnapshot).Return(newSnapshot(), nil).Build()

		state, err := s.currentSession(t.Context())
		So(err, ShouldBeNil)
		So(state, ShouldNotEqual, previous)
		So(build.MockTimes(), ShouldEqual, 1)
	})

	PatchConvey("switched off rounds carry no snapshot", t, func() {
		s := staleServer(&model.SchedulerExtenderConfig{SchedulerExtenderEnabled: false}, nil, nil)
		build := Mock((*Server).buildSnapshot).Return(newSnapshot(), nil).Build()
		s.accumulator.reserve(newView(jobB, rl("cpu", "1")))

		state, err := s.currentSession(t.Context())
		So(err, ShouldBeNil)
		So(state.snap, ShouldBeNil)
		So(build.MockTimes(), ShouldEqual, 0)
		So(s.accumulator.entries, ShouldContainKey, jobB)
	})

	t.Run("close silence", func(t *testing.T) {
		enabled := &model.SchedulerExtenderConfig{SchedulerExtenderEnabled: true}
		PatchConvey("logs once while volcano keeps asking without closing rounds", t, func() {
			s := staleServer(enabled, nil, nil)
			lines := captureLog(s)
			Mock((*Server).buildSnapshot).Return(newSnapshot(), nil).Build()
			s.firstAskSinceClose = fixedNow

			_, err := s.currentSession(t.Context())
			So(err, ShouldBeNil)
			s.session.builtAt = fixedNow
			_, err = s.currentSession(t.Context())
			So(err, ShouldBeNil)
			So(*lines, ShouldHaveLength, 1)
			So((*lines)[0], ShouldContainSubstring, "onSessionClose")
		})
		PatchConvey("stays quiet on the first ask after a close", t, func() {
			s := staleServer(enabled, nil, nil)
			lines := captureLog(s)
			Mock((*Server).buildSnapshot).Return(newSnapshot(), nil).Build()

			_, err := s.currentSession(t.Context())
			So(err, ShouldBeNil)
			So(*lines, ShouldBeEmpty)
			So(s.firstAskSinceClose, ShouldNotBeZeroValue)
		})
		PatchConvey("stays quiet while the plugin switch is off", t, func() {
			s := staleServer(&model.SchedulerExtenderConfig{SchedulerExtenderEnabled: false}, nil, nil)
			lines := captureLog(s)
			s.firstAskSinceClose = fixedNow

			_, err := s.currentSession(t.Context())
			So(err, ShouldBeNil)
			So(*lines, ShouldBeEmpty)
		})
		PatchConvey("a close notification re-arms the log", t, func() {
			s := staleServer(enabled, nil, nil)
			lines := captureLog(s)
			Mock((*Server).buildSnapshot).Return(newSnapshot(), nil).Build()
			s.firstAskSinceClose = fixedNow
			_, err := s.currentSession(t.Context())
			So(err, ShouldBeNil)

			s.invalidateSession()
			So(s.firstAskSinceClose, ShouldBeZeroValue)
			So(s.closeSilenceWarned, ShouldBeFalse)

			s.lastCloseAt = fixedNow
			s.firstAskSinceClose = fixedNow
			_, err = s.currentSession(t.Context())
			So(err, ShouldBeNil)
			So(*lines, ShouldHaveLength, 2)
		})
	})

	t.Run("errors leave the previous round untouched", func(t *testing.T) {
		PatchConvey("config error", t, func() {
			s := staleServer(nil, errors.New("config unavailable"), nil)
			previous := s.session
			state, err := s.currentSession(t.Context())
			So(state, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(s.session, ShouldEqual, previous)
		})
		PatchConvey("quota error", t, func() {
			s := staleServer(&model.SchedulerExtenderConfig{SchedulerExtenderEnabled: true}, nil, errors.New("quota unavailable"))
			previous := s.session
			_, err := s.currentSession(t.Context())
			So(err, ShouldNotBeNil)
			So(s.session, ShouldEqual, previous)
		})
		PatchConvey("snapshot error", t, func() {
			s := staleServer(&model.SchedulerExtenderConfig{SchedulerExtenderEnabled: true}, nil, nil)
			Mock((*Server).buildSnapshot).Return(nil, errors.New("cache unavailable")).Build()
			previous := s.session
			_, err := s.currentSession(t.Context())
			So(err, ShouldNotBeNil)
			So(s.session, ShouldEqual, previous)
		})
	})
}

func TestHandleJobEnqueueable(t *testing.T) {
	PatchConvey("answers the vote as JSON", t, func() {
		stubJobNamespace()
		stubQuota(map[string]string{"cpu": "1"})
		s := seededServer(newSnapshot(newView(jobB, rl("cpu", "1"), admitted), newView(jobA, rl("cpu", "1"))))
		body, err := json.Marshal(jobEnqueueableRequest{Job: vcjobRequest(jobA)})
		So(err, ShouldBeNil)

		rec := httptest.NewRecorder()
		s.handleJobEnqueueable(rec, enqueueRequest(body))
		So(rec.Code, ShouldEqual, http.StatusOK)
		So(rec.Header().Get("Content-Type"), ShouldEqual, "application/json")
		So(strings.TrimSpace(rec.Body.String()), ShouldEqual, `{"status":-1}`)

		rec = httptest.NewRecorder()
		s.handleJobEnqueueable(rec, enqueueRequest([]byte(`{"job":{"Name":"plain-pod-group"}}`)))
		So(rec.Code, ShouldEqual, http.StatusOK)
		So(strings.TrimSpace(rec.Body.String()), ShouldEqual, `{"status":0}`)
	})
}

func TestHandleSessionClose(t *testing.T) {
	PatchConvey("only POST drops the round", t, func() {
		s := seededServer(newSnapshot())
		rec := httptest.NewRecorder()
		s.handleSessionClose(rec, httptest.NewRequest(http.MethodGet, closePath, http.NoBody))
		So(rec.Code, ShouldEqual, http.StatusMethodNotAllowed)
		So(s.session, ShouldNotBeNil)
		So(s.lastCloseAt, ShouldBeZeroValue)

		rec = httptest.NewRecorder()
		s.handleSessionClose(rec, httptest.NewRequest(http.MethodPost, closePath, http.NoBody))
		So(rec.Code, ShouldEqual, http.StatusOK)
		So(s.session, ShouldBeNil)
		So(s.lastCloseAt, ShouldNotBeZeroValue)
	})
}
