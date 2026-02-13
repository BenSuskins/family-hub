package handlers

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bensuskins/family-hub/internal/models"
)

func TestParseIngredientGroups(t *testing.T) {
	tests := []struct {
		name     string
		form     url.Values
		expected []models.IngredientGroup
	}{
		{
			name: "single group",
			form: url.Values{
				"group_name_0":  {"Sauce"},
				"group_items_0": {"1 cup tomato\n2 cloves garlic"},
			},
			expected: []models.IngredientGroup{
				{Name: "Sauce", Items: []string{"1 cup tomato", "2 cloves garlic"}},
			},
		},
		{
			name: "multiple sequential groups",
			form: url.Values{
				"group_name_0":  {"Dough"},
				"group_items_0": {"2 cups flour\n1 tsp salt"},
				"group_name_1":  {"Filling"},
				"group_items_1": {"1 lb cheese"},
			},
			expected: []models.IngredientGroup{
				{Name: "Dough", Items: []string{"2 cups flour", "1 tsp salt"}},
				{Name: "Filling", Items: []string{"1 lb cheese"}},
			},
		},
		{
			name: "no name defaults to Main",
			form: url.Values{
				"group_name_0":  {""},
				"group_items_0": {"3 eggs"},
			},
			expected: []models.IngredientGroup{
				{Name: "Main", Items: []string{"3 eggs"}},
			},
		},
		{
			name:     "empty form returns nil",
			form:     url.Values{},
			expected: nil,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			request, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(testCase.form.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			request.ParseForm()

			result := parseIngredientGroups(request)

			if len(result) != len(testCase.expected) {
				t.Fatalf("expected %d groups, got %d", len(testCase.expected), len(result))
			}

			for i, group := range result {
				if group.Name != testCase.expected[i].Name {
					t.Errorf("group[%d] name: expected %q, got %q", i, testCase.expected[i].Name, group.Name)
				}
				if len(group.Items) != len(testCase.expected[i].Items) {
					t.Errorf("group[%d] items count: expected %d, got %d", i, len(testCase.expected[i].Items), len(group.Items))
					continue
				}
				for j, item := range group.Items {
					if item != testCase.expected[i].Items[j] {
						t.Errorf("group[%d].items[%d]: expected %q, got %q", i, j, testCase.expected[i].Items[j], item)
					}
				}
			}
		})
	}
}
