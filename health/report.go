// SPDX-License-Identifier: MIT
//
// Copyright (c) 2024 Berachain Foundation
//
// Permission is hereby granted, free of charge, to any person
// obtaining a copy of this software and associated documentation
// files (the "Software"), to deal in the Software without
// restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following
// conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package health

import (
	"context"
	"fmt"
	"time"
)

// reportingInterval is the interval at which the health service
// logs the health status of services.
const reportingInterval = 10 * time.Second

// reportingLoop initiates a loop that periodically checks and
// reports the health status of services.
func (s *Service) reportingLoop(ctx context.Context) {
	ticker := time.NewTicker(reportingInterval)
	for {
		select {
		case <-ticker.C:
			s.reportStatuses()
		case <-ctx.Done():
			return
		}
	}
}

// reportStatuses logs the health status of all services.
func (s *Service) reportStatuses() {
	var unhealthy []string
	for _, svc := range s.retrieveStatuses() {
		if !svc.Healthy {
			unhealthy = append(
				unhealthy,
				fmt.Sprintf("%s: %s", svc.Name, svc.Err),
			)
		}
	}

	if len(unhealthy) == 0 {
		s.Logger().Info("all beacon services are reporting healthy 🌤️ ")
	} else {
		s.Logger().Error("unhealthy services detected 🌧️ ", "statuses", unhealthy)
	}
}
