package models

import "time"

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

type ChoreStatus string

const (
	ChoreStatusPending   ChoreStatus = "pending"
	ChoreStatusCompleted ChoreStatus = "completed"
	ChoreStatusOverdue   ChoreStatus = "overdue"
)

type RecurrenceType string

const (
	RecurrenceNone     RecurrenceType = "none"
	RecurrenceDaily    RecurrenceType = "daily"
	RecurrenceWeekly   RecurrenceType = "weekly"
	RecurrenceMonthly  RecurrenceType = "monthly"
	RecurrenceCustom   RecurrenceType = "custom"
	RecurrenceCalendar RecurrenceType = "calendar"
)

type AssignmentStatus string

const (
	AssignmentStatusAssigned   AssignmentStatus = "assigned"
	AssignmentStatusCompleted  AssignmentStatus = "completed"
	AssignmentStatusReassigned AssignmentStatus = "reassigned"
)

type User struct {
	ID          string
	OIDCSubject string
	Email       string
	Name        string
	AvatarURL   string
	Role        Role
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Category struct {
	ID              string
	Name            string
	CreatedByUserID string
	CreatedAt       time.Time
}

type Chore struct {
	ID              string
	Name            string
	Description     string
	CreatedByUserID string
	CategoryID      *string

	AssignedToUserID   *string
	LastAssignedIndex  int
	EligibleAssignees  []string

	DueDate *time.Time
	DueTime *string

	RecurrenceType  RecurrenceType
	RecurrenceValue string
	RecurOnComplete bool

	Status          ChoreStatus
	CompletedAt     *time.Time
	CompletedByUserID *string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type Event struct {
	ID              string
	Title           string
	Description     string
	Location        string
	StartTime       time.Time
	EndTime         *time.Time
	AllDay          bool
	CategoryID      *string
	CreatedByUserID string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ChoreAssignment struct {
	ID          string
	ChoreID     string
	UserID      string
	AssignedAt  time.Time
	CompletedAt *time.Time
	Status      AssignmentStatus
}

type APIToken struct {
	ID              string
	Name            string
	TokenHash       string
	CreatedByUserID string
	ExpiresAt       *time.Time
	CreatedAt       time.Time
}

type IngredientGroup struct {
	Name  string   `json:"name"`
	Items []string `json:"items"`
}

type Recipe struct {
	ID              string
	Title           string
	Instructions    string
	Ingredients     []IngredientGroup
	Servings        *int
	PrepTime        *string
	CookTime        *string
	SourceURL       *string
	CategoryID      *string
	CreatedByUserID string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type MealType string

const (
	MealTypeBreakfast MealType = "breakfast"
	MealTypeLunch     MealType = "lunch"
	MealTypeDinner    MealType = "dinner"
)

type MealPlan struct {
	Date            string
	MealType        MealType
	RecipeID        *string
	Name            string
	Notes           string
	CreatedByUserID string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
