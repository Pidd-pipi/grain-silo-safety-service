package main

import (
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
)

const opsDomainName = "grain-silo-safety-service"

type OpsStatus string

const (
	OpsStatusQueued OpsStatus = "queued"
	OpsStatusActive OpsStatus = "active"
	OpsStatusPaused OpsStatus = "paused"
	OpsStatusClosed OpsStatus = "closed"
)

type OpsPriority string

const (
	OpsPriorityLow      OpsPriority = "low"
	OpsPriorityNormal   OpsPriority = "normal"
	OpsPriorityHigh     OpsPriority = "high"
	OpsPriorityCritical OpsPriority = "critical"
)

type OpsRecord struct {
	ID        string            `json:"id"`
	Subject   string            `json:"subject"`
	Owner     string            `json:"owner"`
	Status    OpsStatus         `json:"status"`
	Priority  OpsPriority       `json:"priority"`
	Revision  int               `json:"revision"`
	Labels    map[string]string `json:"labels"`
	CreatedAt string            `json:"createdAt"`
	UpdatedAt string            `json:"updatedAt"`
}

type OpsRule struct {
	Code           string
	Name           string
	Severity       OpsPriority
	RequiredLabels []string
	Terminal       bool
}

type OpsEvent struct {
	ID       string            `json:"id"`
	RecordID string            `json:"recordId"`
	Type     string            `json:"type"`
	Actor    string            `json:"actor"`
	At       string            `json:"at"`
	Details  map[string]string `json:"details"`
}

type OpsQuery struct {
	Subject  string
	Status   OpsStatus
	Priority OpsPriority
	Owner    string
	Page     int
	PageSize int
}

type OpsPage struct {
	Items    []OpsRecord `json:"items"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
	Total    int         `json:"total"`
	HasNext  bool        `json:"hasNext"`
}

type OpsSnapshot struct {
	Domain      string              `json:"domain"`
	GeneratedAt string              `json:"generatedAt"`
	Records     int                 `json:"records"`
	Active      int                 `json:"active"`
	ByStatus    map[OpsStatus]int   `json:"byStatus"`
	ByPriority  map[OpsPriority]int `json:"byPriority"`
}

func (r OpsRecord) Clone() OpsRecord {
	copy := r
	copy.Labels = map[string]string{}
	for key, value := range r.Labels {
		copy.Labels[key] = value
	}
	return copy
}

func (r OpsRecord) LabelValue(key string) string { return r.Labels[key] }
func (r OpsRecord) Terminal() bool               { return r.Status == OpsStatusClosed }

var opsRecordSequence = uint64(100)

func newOpsRecordID() string { return fmt.Sprintf("wo-%03d", atomic.AddUint64(&opsRecordSequence, 1)) }

func (p OpsPriority) Weight() int {
	switch p {
	case OpsPriorityCritical:
		return 4
	case OpsPriorityHigh:
		return 3
	case OpsPriorityNormal:
		return 2
	default:
		return 1
	}
}

func normalizeOpsRecord(record OpsRecord) OpsRecord {
	record.ID = strings.ToLower(strings.TrimSpace(record.ID))
	record.Subject = strings.Join(strings.Fields(record.Subject), " ")
	record.Owner = strings.TrimSpace(record.Owner)
	if record.Revision < 1 {
		record.Revision = 1
	}
	if record.Labels == nil {
		record.Labels = map[string]string{}
	}
	return record
}

func sortOpsRecords(items []OpsRecord) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Priority.Weight() != items[j].Priority.Weight() {
			return items[i].Priority.Weight() > items[j].Priority.Weight()
		}
		return items[i].UpdatedAt > items[j].UpdatedAt
	})
}

func opsRules() []OpsRule {
	out := make([]OpsRule, 0, 112)
	for _, group := range [][]OpsRule{
		opsRules01(), opsRules02(), opsRules03(), opsRules04(), opsRules05(), opsRules06(), opsRules07(),
		opsRules08(), opsRules09(), opsRules10(), opsRules11(), opsRules12(), opsRules13(), opsRules14(),
	} {
		out = append(out, group...)
	}
	return out
}
