/*
Copyright 2024 Adobe. All rights reserved.
This file is licensed to you under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License. You may obtain a copy
of the License at http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under
the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
OF ANY KIND, either express or implied. See the License for the specific language
governing permissions and limitations under the License.
*/

package webhook

import "time"

const (
	// AlertStatusFiring alert status when it is firing
	AlertStatusFiring string = "firing"

	// AlertStatusResolved alert status when it is resolved
	AlertStatusResolved string = "resolved"
)

// Alert represents an Alertmanager alert
type Alert struct {
	Receiver          string            `json:"receiver"`
	Status            string            `json:"status"`
	Alerts            []AlertItem       `json:"alerts"`
	GroupLabels       GroupLabels       `json:"groupLabels"`
	CommonLabels      CommonLabels      `json:"commonLabels"`
	CommonAnnotations CommonAnnotations `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
}

// AlertItem ...
type AlertItem struct {
	Status       string            `json:"status"`
	Labels       AlertLabels       `json:"labels"`
	Annotations  CommonAnnotations `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

// AlertLabels ...
type AlertLabels struct {
	Alertname string `json:"alertname"`
	Service   string `json:"service"`
	Severity  string `json:"severity"`
}

// GroupLabels ...
type GroupLabels struct {
	Alertname string `json:"alertname"`
}

// CommonLabels ...
type CommonLabels struct {
	Alertname string `json:"alertname"`
	Service   string `json:"service"`
	Severity  string `json:"severity"`
}

// CommonAnnotations ...
type CommonAnnotations struct {
	Summary string `json:"summary"`
}
