package domain_test

import (
	"testing"

	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestPriorityFromInt(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want domain.TaskPriority
	}{
		{name: "zero is low", in: 0, want: domain.TaskPriorityLow},
		{name: "1 is low", in: 1, want: domain.TaskPriorityLow},
		{name: "3 is low", in: 3, want: domain.TaskPriorityLow},
		{name: "4 is mid", in: 4, want: domain.TaskPriorityMid},
		{name: "7 is mid", in: 7, want: domain.TaskPriorityMid},
		{name: "8 is high", in: 8, want: domain.TaskPriorityHigh},
		{name: "10 is high", in: 10, want: domain.TaskPriorityHigh},
		{name: "100 is high", in: 100, want: domain.TaskPriorityHigh},
		{name: "negative is low", in: -1, want: domain.TaskPriorityLow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, domain.PriorityFromInt(tt.in))
		})
	}
}

func TestNewTaskCategory(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want domain.TaskCategory
	}{
		{name: "work", in: "work", want: domain.TaskCategoryWork},
		{name: "study", in: "study", want: domain.TaskCategoryStudy},
		{name: "personal", in: "personal", want: domain.TaskCategoryPersonal},
		{name: "unknown defaults to personal", in: "garbage", want: domain.TaskCategoryPersonal},
		{name: "empty defaults to personal", in: "", want: domain.TaskCategoryPersonal},
		{name: "case-sensitive: WORK is unknown", in: "WORK", want: domain.TaskCategoryPersonal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, domain.NewTaskCategory(tt.in))
		})
	}
}

func TestTaskCategory_StringRu(t *testing.T) {
	tests := []struct {
		name string
		in   domain.TaskCategory
		want string
	}{
		{name: "work", in: domain.TaskCategoryWork, want: "Работа"},
		{name: "study", in: domain.TaskCategoryStudy, want: "Учеба"},
		{name: "personal", in: domain.TaskCategoryPersonal, want: "Личное"},
		{name: "unknown", in: domain.TaskCategory("xxx"), want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.in.StringRu())
		})
	}
}

func TestTaskID_String(t *testing.T) {
	assert.Equal(t, "abc", domain.TaskID("abc").String())
	assert.Equal(t, "", domain.TaskID("").String())
}
